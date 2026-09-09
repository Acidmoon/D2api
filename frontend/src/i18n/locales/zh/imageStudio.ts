export default {
  imageStudio: {
    title: '生图创作中心',
    description: '用你自己的 API Key 生成图片，可放大预览并下载原图。',
    key: {
      label: '创作分组 API Key',
      placeholder: '选择 API Key',
      hint: '所属分组：{group}。本次生成的费用计入该 Key。',
      empty: '没有已开启生图权限的 API Key，请先到「API 密钥」页创建。',
      refresh: '刷新',
      create: '创建 API Key'
    },
    form: {
      model: '模型',
      modelPlaceholder: '选择模型',
      modelsLoading: '正在加载模型…',
      modelsEmpty: '该分组暂无可用模型。',
      prompt: '提示词',
      promptPlaceholder: '描述你想要的画面…',
      size: '尺寸',
      sizeAuto: '自动',
      count: '张数',
      generate: '生成',
      generating: '生成中…',
      costHint: '生成按张计费，费用记入所选分组。'
    },
    result: {
      title: '生成结果',
      clear: '清空',
      emptyTitle: '还没有图片',
      emptyDescription: '在左侧填写提示词后点击「生成」。',
      noKeyTitle: '没有可用的生图 Key',
      noKeyDescription: '请先在一个开启了生图权限的分组下创建 API Key。',
      loading: '正在生成，可能需要一会儿…',
      open: '查看',
      download: '下载',
      downloading: '下载中…',
      downloadOriginal: '下载原图',
      revisedPrompt: '修订后的提示词',
      generated: '已生成 {count} 张图片'
    },
    lightbox: {
      title: '预览'
    },
    errors: {
      keyRequired: '请先选择 API Key。',
      modelRequired: '请先选择模型。',
      promptRequired: '请先填写提示词。',
      modelsFailed: '加载模型失败：{message}',
      generateFailed: '生成失败：{message}',
      downloadFailed: '下载失败：{message}',
      emptyResult: '上游没有返回图片。'
    }
  }
}
