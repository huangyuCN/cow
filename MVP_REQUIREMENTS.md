# 问题描述
许多服务端业务，尤其是游戏服业务，具备以下特征：
1.某个聚合根（如玩家、房间、会话）长期常驻于进程内存。
2.对该聚合的修改由单个 goroutine 串行处理，或由宿主提供等价的串行保证。
3.一次请求、一次消息处理、一次显式调用，天然形成清晰的处理作用域。
在这个前提下，业务虽然不需要解决同一聚合的并发写冲突，但仍然会遇到两个长期问题：
1.处理链变长后，中途失败会让补偿逻辑变得脆弱、难验证。补偿路径易漏、调试困难、心智负担高。
2.直接做整对象深拷贝虽然简单，但在大对象、map、slice 较多时成本过高。CPU、alloc、GC 成本高，大对象下放大明显。
有没有好的思路能解决这个问题，最好是能够像整体拷贝一样简单，但是CPU和内存消耗不能增加太多。如果有必要，可以通过像k8s Deepcopy一样，自动代码生成一些代码，来提升效率，减少心智负担。


这是一份MVP（最小可行性验证）规格需求文档。
------------------------------
## PRD: 单协程大对象“操作留痕与增量回滚”MVP 验证方案
## 1. 背景与核心痛点 (Context & Problem)
在游戏服等单线程/协程串行处理聚合根的业务场景中，对象体积大且包含大量不确定深度的嵌套 Map 和 Slice。

* 痛点 1（脆弱的补偿）：处理链变长后，中途失败的业务补偿逻辑极易漏写，心智负担高。
* 痛点 2（深拷贝成本高）：全量 Deepcopy 虽然安全，但在大对象、多 Map/Slice 场景下，CPU、内存分配（alloc）和 GC 压力巨大。
* 终极目标：实现类似数据库事务的 Rollback 能力。业务报错时一键自动回滚，同时保持接近零的内存分配与极低的 CPU 开销。

## 2. 核心架构设计 (Core Architecture)
本方案采用“操作留痕与逆操作撤销（Undo Log / Command Pattern）”架构。

* 核心原理：不拷贝数据，只拷贝“动作的逆操作”。在单协程串行环境下，利用一个复用的事务上下文（TxContext）收集所有写操作的逆向恢复函数。
* 成功场景：执行完毕，清空上下文，零内存拷贝，直接提交。
* 失败场景：倒序执行上下文中的所有 Undo 函数，瞬间恢复原始状态。

------------------------------
## 3. 核心组件定义 (Component Specification)
## 3.1 事务上下文 (TxContext)

* 职责：作为单次请求/消息处理作用域（Scope）的生命周期管理者，收集并执行 Undo 闭包。
* 优化要求：必须支持 Reset，通过 sync.Pool 进行对象池化复用，确保高频调用下 GC 动作为 0。

## 3.2 聚合根数据结构 (Data Model)
作为 MVP 验证，定义一个包含复杂嵌套、Map、Slice，且同时带有 Protobuf 和 MongoDB BSON 标签的模拟玩家结构体 Player。
## 3.3 自动化生成的变更代理 API (Generated Proxy APIs)
通过代码生成（本 MVP 阶段先手写模拟）为 Player 及其子结构体提供安全的写操作代理方法：

   1. Slice 代理：
   * Append 操作：记录操作前的 len，回滚时直接利用切片截断（slice = slice[:oldLen]），不触发扩容和底层数组拷贝。
      * Set 修改操作：记录被替换位置旧元素的指针/值，回滚时原位覆盖。
   2. Map 代理：
   * Put 操作：检查 Key 是否存在。若存在，记录旧值；若不存在，记录“回滚时删除该 Key”。
   3. 嵌套指针（Pointer）代理：
   * GetForWrite 操作：采用延迟局部深拷贝（Lazy Local Copy）。只有当深层子结构需要被修改时，才对该子结构执行 Clone() 替换，并将老指针压入 Undo 栈。若回滚，直接将老指针挂回原位。

------------------------------
## 4. MVP 验证代码实现框架 (Implementation Template)
请根据以下技术规格，生成完整的 Go 验证代码。代码需要包含完整的 main 函数，并使用 testing.B 或时间戳打印出成功提交与失败回滚两种场景下的耗时与 Runtime 内存分配指标。

package main
import (
	"fmt"
	"sync"
)
// ==========================================// 1. 基础定义与标签支持 (同时兼容 PB 与 BSON)// ==========================================
type Item struct {
	Id   int64  `protobuf:"varint,1,opt,name=id" json:"id,omitempty" bson:"_id"`
	Name string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty" bson:"name"`
}
type Hero struct {
	HeroId int32 `protobuf:"varint,1,opt,name=hero_id" json:"hero_id,omitempty" bson:"hero_id"`
	Level  int32 `protobuf:"varint,2,opt,name=level" json:"level,omitempty" bson:"level"`
}
// Clone 实现单层深拷贝 (用于延迟局部拷贝)func (h *Hero) Clone() *Hero {
	if h == nil {
		return nil
	}
	return &Hero{HeroId: h.HeroId, Level: h.Level}
}
type Player struct {
	Uid    int64             `protobuf:"varint,1,opt,name=uid" json:"uid,omitempty" bson:"_id"`
	Assets map[string]int64  `protobuf:"bytes,2,rep,name=assets" json:"assets,omitempty" bson:"assets"`
	Items  []*Item           `protobuf:"bytes,3,rep,name=items" json:"items,omitempty" bson:"items"`
	Hero   *Hero             `protobuf:"bytes,4,opt,name=hero" json:"hero,omitempty" bson:"hero"` // 嵌套指针
}
// ==========================================// 2. 事务上下文实现 (TxContext)// ==========================================
type TxContext struct {
	undoLogs []func()
}
func (ctx *TxContext) AddUndo(undo func()) {
	ctx.undoLogs = append(ctx.undoLogs, undo)
}
func (ctx *TxContext) Rollback() {
	// 必须倒序执行
	for i := len(ctx.undoLogs) - 1; i >= 0; i-- {
		ctx.undoLogs[i]()
	}
}
func (ctx *TxContext) Reset() {
	// 复用底层切片内存，防止 GC
	ctx.undoLogs = ctx.undoLogs[:0]
}
var txPool = sync.Pool{
	New: func() interface{} {
		return &TxContext{undoLogs: make([]func(), 0, 16)}
	},
}
// ==========================================// 3. 代理方法 (待 AI 补全的具体生成逻辑)// ==========================================
// TODO: AI 请补全以下三个核心代理函数// func (p *Player) PutAsset(ctx *TxContext, key string, val int64)// func (p *Player) AppendItem(ctx *TxContext, item *Item)// func (p *Player) GetHeroForWrite(ctx *TxContext) *Hero
// ==========================================// 4. 业务场景验证 (main)// ==========================================
func main() {
	// 初始化一个超大的常驻内存 Player 对象
	player := &Player{
		Uid:    10001,
		Assets: map[string]int64{"gold": 1000, "diamond": 100},
		Items:  []*Item{{Id: 1, Name: "Sword"}},
		Hero:   &Hero{HeroId: 99, Level: 1},
	}

	fmt.Printf("【初始状态】: %+v, Hero: %+v\n", player, player.Hero)

	// --- 场景 A: 模拟业务处理中途报错，触发自动回滚 ---
	errScenario := func() error {
		ctx := txPool.Get().(*TxContext)
		ctx.Reset()
		defer func() {
			// 如果有错误，一键回滚
			ctx.Rollback()
			txPool.Put(ctx)
		}()

		// 模拟业务操作
		// 1. 修改 Map
		// player.PutAsset(ctx, "gold", 500)
		// 2. 追加 Slice
		// player.AppendItem(ctx, &Item{Id: 2, Name: "Shield"})
		// 3. 修改嵌套指针 (延迟局部深拷贝)
		// h := player.GetHeroForWrite(ctx)
		// h.Level = 2

		// 模拟中途发生逻辑错误
		return fmt.Errorf("something went wrong in business chain")
	}()

	fmt.Printf("【回滚后状态 (预期与初始一致)】: %+v, Hero: %+v, Err: %v\n", player, player.Hero, errScenario)

	// --- 场景 B: 模拟业务成功提交 ---
	// ... 补全成功场景验证 ...
}

------------------------------
## 5. 对 AI 提示词与指令 (AI Prompts)
请遵照以下指令完成代码编写：

   1. 严格契合单协程前提：由于宿主提供单协程串行保证，TxContext 内部的 undoLogs 切片不需要加锁（Mutex），追求极致的单线程性能。
   2. 实现三个写代理函数：按照第 3.3 节的逻辑，完美补全模板中 TODO 的三个代理方法。注意 Map 的 Key 不存在时回滚需要 delete；Slice 回滚使用高性能的切片截断。
   3. 补充基准性能测试（Benchmark）：在生成的文件中，提供一个基准测试函数（或在 main 中循环 100,000 次），对比本方案与“每次请求都对 Player 执行全量 Deepcopy”在 CPU 耗时和 Mallocs/op 上的巨大优势。

