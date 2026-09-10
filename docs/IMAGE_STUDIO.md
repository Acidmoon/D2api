# 生图创作中心（Image Studio）

用户端页面 `/image-studio`（`frontend/src/views/user/ImageStudioView.vue`），让用户用自己
「创作分组」下的 API Key 直接调网关 `/v1/images/generations` 生图，并把结果保存到
7 天有效期的个人历史（`docs` 见 `backend/internal/service/image_studio_history.go`）。

## 组成

| 层 | 文件 | 说明 |
|---|---|---|
| 网关客户端 | `frontend/src/api/imageStudio.ts` | `/v1/models`、`/v1/images/generations`，携带用户 Key（`Authorization: Bearer <key>`），不走用户会话 REST |
| 历史 | `frontend/src/api/imageStudioHistory.ts` | 走 `apiClient`（JWT），服务端按 `user_id` 隔离 |
| 入口门控 | `frontend/src/composables/useImageStudioAccess.ts` | 只认分组名含「创作」且 `allow_image_generation` 的 Key；`primary_group` 优先，`group` 兜底 |
| 参数逻辑 | `frontend/src/composables/useImageStudioOptions.ts` | 纯函数：模型过滤、尺寸档位、单价/预估、档位归一、金额格式化 |
| 计费落档 | `backend/internal/service/image_output_accounting.go`、`image_billing_size.go` | 解析真实输出尺寸 → 决定 1K/2K/4K 档 |

## 上游能力实测（2026-09-10，生产 group 30「创作分组」，上游 wegoapi）

用临时 Key 打真实网关 + 直连上游各跑一轮，结论：

1. `GET /v1/models` 在该分组只返回 4 个模型：`gpt-image-2`、`gpt-image-2.5`、
   `gpt-image-2.5-flare`、`gpt-image-2.5-sunburst`。分组白名单里写的
   `gpt-image-1` / `gpt-image-1.5` 上游并没有，选中即 `404 model_not_found`。
   → 页面模型下拉只列 `gpt-image-*` / `grok-imagine*`（判据与后端
   `isOpenAIImageGenerationModel` 一致），并提示隐藏了多少个非生图模型。
2. `size` **被上游回显但不被执行**：`1K` → 1254×1254、`2K` → 1536×1024、
   `4K` → 1448×1086，全部约 1.57MP；`2048x2048`、`1x1`、甚至非法值也一样出图。
3. `quality`（low/high）、`output_format`（请求 jpeg 回来的仍是 PNG）、
   `aspect_ratio`（请求 9:16 拿到 1.25:1）都是**假开关**。
   → 界面上不放这三项，放了就是给用户看得见点不动的选项。

## 计费落档（重要）

`/v1/images/generations` 的 APIKey 转发路径此前只认上游 JSON 里的 `data[].size` 字段；
上述上游不回传该字段，于是 `ImageOutputSizes` 为空，
`ApplyOpenAIImageBillingResolution` 回退成**按请求值落档**
（`usage_logs.image_size_source = 'input'`）。后果：客户端把 `size` 写成 `4K` 或任意
非法值，就用 4K 单价买了一张实际 ~1.6MP 的图。

现在 `addDataArray` 在 `size` 缺失或为 `auto` 时，用 `detectOpenAIImageResultSize`
从图片字节头解出真实宽高（与 OAuth/responses 路径
`reconcileOpenAIImageResultSizes` 同一手法，只读头部几十字节），于是：

- `image_size` 按真实输出落档，`image_size_source = 'output'`，
  `image_output_size` 有值；
- 请求档位不再能操纵单价。

**价格影响需要知晓**：该上游恒定输出 ~1.57MP（长边 1254–1536），按平台自身的
档位规则（长边 ≤1024 为 1K，≤2048 为 2K）属于 **2K 档**。因此修好之后，
该分组的每张图都会落 2K 价，而不是随请求的 1K/4K 变化。这与分组 24 已有的
chat 路径出图计费方式一致（那里 `image_size=2K, source=output`）。

界面因此不再宣称「选大档位=更清晰」：尺寸区显示每个档位在当前分组下的
**单张价**与预估总价，并在每张结果图左上角标注**实测像素**（读
`img.naturalWidth/Height`，不信上游回显）。

## 维护要点

- 上游换供应商或真的支持 2K/4K 时，先按上面的方法实测再调档位常量
  （`IMAGE_SIZE_TIERS`），不要凭 OpenAI 官方文档假设。
- 新增图像模型族时，同时改 `isImageGenerationModel`（前端）与
  `isOpenAIImageGenerationModel`（后端），两边判据必须一致，否则会出现
  「界面能选、网关 400」。
- 尺寸档位持久化在 localStorage（`image_studio_size`）；
  `normalizeSizeTier` 会把旧的像素写法归一成档位，改档位集合时留意历史值。
- 取消生成只是断开前端连接；上游可能已经出图并计费，提示文案里已说明。
