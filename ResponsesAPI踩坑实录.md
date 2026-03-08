# ResponsesAPI踩坑实录

## 1. 为什么要写这篇

这个仓库原本已经有一套基于 Chat Completions 的 OpenAI 兼容封装：

- `pkg/llm_core/client/openai`

后来又加了一套基于 OpenAI 官方 SDK `github.com/openai/openai-go` 的 Responses API 封装：

- `pkg/llm_core/client/openai_official`

纸面上看，两者都能做文本、多轮对话、function calling、streaming；但真正接上不同 API 中转站之后，会发现 Responses API 对 schema、输入形态、网关兼容性的要求明显更严格。这篇就是把这次踩过的坑完整记下来。

## 2. Responses API 和 Chat Completions 最大的差异

### 2.1 请求入口不同

- Chat Completions：`/chat/completions`
- Responses API：`/responses`

很多中转站会说自己“兼容 OpenAI”，但实际只兼容 Chat Completions，不一定兼容 `/responses`。

### 2.2 输入结构不同

Chat Completions 的经典输入是：

```json
{
  "model": "...",
  "messages": [
    {"role": "system", "content": "..."},
    {"role": "user", "content": "..."}
  ]
}
```

Responses API 的输入更灵活，可以是字符串，也可以是输入 item 列表，但在网关世界里，“灵活”往往会变成“兼容性不稳定”。这个仓库最后实际采用的是更稳的标准 item 列表形态。

在 `pkg/llm_core/client/openai_official/utils.go` 里，当前封装统一把消息转成 Responses 的 input items，而不是继续沿用 `messages`。

### 2.3 输出结构不同

Chat Completions 常见输出是：

- `choices[0].message.content`
- `choices[0].message.tool_calls`

Responses API 则是：

- `output[]`
- item 可能是 `message`
- item 也可能是 `function_call`
- 流式事件还是更细粒度的 event union

这也是为什么 `pkg/llm_core/client/openai_official/stream.go` 比旧版 chat completions 的流式拼装更复杂。

## 3. 为什么 Responses API 更容易踩坑

一句话：它更现代，也更严格。

更现代，意味着：

- 抽象更统一
- tool call、message、reasoning item 都能放进统一 output 体系
- streaming event 更细

更严格，意味着：

- function schema 更容易被校验
- tool choice 形态更多
- gateway 只要少兼容一部分，立刻就会报错

所以 Chat Completions 经常是“怎么传都能凑合跑”，Responses API 则更像“稍微不对就直接拒绝”。

## 4. 这次实际踩到的第一个坑：strict schema 不是随便开的

在 `pkg/llm_core/client/openai_official` 里，一开始 function tool 一律按 strict schema 发出。结果接 `gpt-5.4` 时，只要 schema 不够闭合，就直接被拒。

典型坑点：

- 开了 `strict=true`
- 却没有补 `parameters.additionalProperties=false`

这会直接触发：

- `invalid_function_parameters`

后来修复时发现：只补 `additionalProperties=false` 还不够。

如果你的 schema 里存在 optional 参数，而仍然开 `strict=true`，有些网关会继续拒绝，因为它要求 `properties` 中的字段全部出现在 `required` 里。

因此这个仓库最终做成了：

- 全部字段必填 -> `strict: true`
- 存在 optional 字段 -> `strict: false`

## 5. 第二个坑：nil 不能当作空 schema 发出去

Go 里很容易写出：

- `Properties: nil`
- `Required: nil`

如果直接序列化，很多时候会变成：

- `properties: null`
- `required: null`

但对于 JSON Schema 来说，这通常不是你真正想表达的含义。零参数工具应该是：

- `properties: {}`
- `required: []`

这个问题在 `generate_uuid()` 这类零参数工具上很典型。后来在 `pkg/llm_core/client/openai_official/utils.go` 里做了规范化，把 nil 统一转成空 map / 空数组。

## 6. 第三个坑：中转站的“兼容 OpenAI”不等于兼容 Responses API

这是这次最有价值的实战结论。

### 6.1 OpenAI 官方 / `aihubmix`

这两类端点对 Responses API 的支持相对完整。至少在这个仓库里验证过：

- 标准文本请求可用
- function calling 可用
- tool schema 校验行为稳定

### 6.2 `packyapi`

`packyapi` 不是完全不支持 Responses API，而是“部分兼容”。它的问题不是单一 bug，而是一组兼容性缺口。

实测结论已经写进 `pkg/llm_core/client/README.md`，这里再总结一遍：

- `gpt-5.4` 在 `packyapi` 上要求 `input` 使用标准消息数组形式
- `input: "..."` 可能直接报 `400 bad_response_status_code`
- 显式传 `temperature` / `top_p` 时，`gpt-5.4` 会报 `400 bad_response_status_code`
- 命名函数对象形式的 `tool_choice` 会触发服务端 JSON 反序列化错误
- `/chat/completions` 在这个网关上甚至可能是 `404`

也就是说，某个网关可能：

- 模型列表正常
- `/responses` 也存在
- 简单文本还能通
- 但一加 sampling / tool choice / schema 细节就炸

## 7. 第四个坑：报错表面像 502，根因其实是 schema 或兼容层

这次最开始看到的是类似：

- `502 bad gateway`

但继续往下查会发现，底层很多时候不是模型坏了，而是上游已经返回了更明确的 400 错误，只是被中间代理或网关包装掉了。

常见真实根因包括：

- invalid schema
- 某字段类型不兼容
- gateway 没实现某个 Responses API 子能力

经验上，如果你只盯着最外层 502，很容易误判成“模型不稳定”；更稳妥的做法是直接做 raw HTTP 对照测试，把同一个 payload 分别打到：

- 官方端点
- 中转站 A
- 中转站 B

这样很快就能看出问题到底出在 wrapper、模型、还是 gateway。

## 8. 第五个坑：Responses API 的 tool choice 兼容面更窄

Chat Completions 时代，很多兼容网关对 `tool_choice` 的容忍度更高。

但到了 Responses API，`tool_choice` 可能有：

- `auto`
- `none`
- `required`
- 指定函数对象

而某些网关只实现了字符串分支，没有实现对象分支。于是你会看到离谱报错，比如：

- 服务端把 `tool_choice` 当 string 解析
- 你发了 object
- 然后它返回 JSON 反序列化错误

这不是 OpenAI 官方协议本身的问题，而是网关实现不完整。

## 9. 第六个坑：Responses API 和 MCP 组合时，schema 一致性比以前更重要

这次还有一个非常容易忽略的点：

- LLM 看到的 function schema
- MCP server 运行时实际接受的参数规则

两边必须一致。

如果你对模型说：

- `additionalProperties: false`
- `required: ["name"]`

那 MCP runtime 就不能再默默接受未声明参数，也不能对缺失必填参数装作没事。

否则就会出现：

- 模型侧 schema 很严格
- 工具侧 runtime 很宽松
- 调试时特别混乱

这也是这次在 `pkg/mcp/model/tool.go` 和 `pkg/mcp/server/server.go` 继续收尾的原因。

## 10. 在这个仓库里，我建议怎么选 API

### 场景一：你主要追求“兼容网关多、坑少”

优先考虑：

- `pkg/llm_core/client/openai`

因为 Chat Completions 的兼容性历史更长，很多中转站先做的是这一层。

### 场景二：你想做更规范的 Responses / streaming / function item 处理

优先考虑：

- `pkg/llm_core/client/openai_official`

但前提是：

- 你确认目标网关真的兼容 `/responses`
- 你愿意为 gateway 差异做一些适配

### 场景三：你要接多个第三方中转站

建议：

1. 先保留 raw HTTP 最小复现脚本
2. 对每个 gateway 单独测文本、tools、sampling、streaming
3. 不要假设“模型列表能出来”就等于“Responses API 没问题”

## 11. 一份实战排错清单

如果 Responses API 出错，我现在会按这个顺序排查：

1. endpoint 是不是 `/responses`
2. gateway 是否真的支持 Responses API
3. `input` 形态是不是 gateway 接受的那种
4. 是否显式传了 `temperature` / `top_p`
5. tool schema 是否补了 `additionalProperties:false`
6. optional 参数是否还错误地开着 `strict=true`
7. zero-arg tool 是否发成了 `required:null` / `properties:null`
8. `tool_choice` 是不是用了 gateway 不支持的对象形态
9. 对照 raw HTTP 和 SDK wrapper，确认是网关问题还是本地组包问题

## 12. 一句话总结

Chat Completions 更像“老江湖接口”，兼容性广但语义偏旧；Responses API 更像“新一代正式接口”，结构更统一、能力更强，但对 schema 和 gateway 兼容性的要求也明显更高。真正踩坑时，问题往往不是“模型不行”，而是“你以为兼容 OpenAI 的网关，其实只兼容了一半”。
