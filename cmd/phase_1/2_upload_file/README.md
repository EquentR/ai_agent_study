# 附件上传调用示例

通过 `llm_core` 向 LLM 发送图片/文本附件。

## 运行方式

```bash
go run ./cmd/phase_1/2_upload_file \
  --question "请结合附件回答" \
  --image ./examples/demo.png \
  --text ./examples/demo.txt
```

可选参数：
- `--model`：模型名（默认 `kimi-k2.5`）
- `--image`：图片附件路径
- `--text`：文本附件路径
