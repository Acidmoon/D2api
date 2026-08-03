export default {
  fingerprint: {
    title: '模型指纹检测',
    description: '对系统账号或外部端点做模型身份核验：比对一词答案的统计分布，判定所供模型与声明模型是否一致',
    // §7.4 判定原理（一句话版）
    help: '判定原理：我们让上游回答「说一个 1–100 的随机数」这类一词问题并重复采样。每个模型回答这类问题时都有自己稳定的「口癖」（偏好分布），不同模型的口癖差别很大。分数 s 衡量被测对象回答的口癖与正版模型的口癖相差多远：0 为完全一致，越大越可疑。',

    create: {
      sectionTitle: '发起检测',
      tabAccount: '测账号',
      tabExternal: '测外部端点',
      account: '被测账号',
      accountPlaceholder: '选择系统内账号',
      model: '模型名',
      modelPlaceholder: '被测对象声明的模型，如 gpt-4o',
      referenceMode: '参考基准',
      referenceModeExisting: '用已有参考',
      referenceModeEnroll: '选可信账号现场注册',
      referenceExisting: '已有参考',
      referenceExistingPlaceholder: '选择已注册的参考基准',
      referenceAccount: '参考账号',
      referenceAccountPlaceholder: '选择可信账号（先注册参考再检测）',
      referenceEnrollHint: '将先用参考账号为该模型注册参考基准，再检测被测对象',
      baseUrl: 'BaseURL',
      baseUrlPlaceholder: 'https://api.example.com',
      apiKey: 'API Key',
      apiKeyPlaceholder: '仅用于本次检测',
      apiKeyHint: 'API Key 只在任务运行期持有，不写入任何文件、不进日志',
      provider: 'Provider',
      apiMode: 'API 模式',
      keepRaw: '保留原始样本',
      keepRawHint: '在报告中附加每个探测项的原始回答，便于人工核查（报告文件体积会明显变大）',
      submit: '发起检测',
      submitting: '发起中…',
      created: '检测任务已发起，进度见下方检测记录',
      createFailed: '发起检测失败',
      accountRequired: '请选择被测账号',
      modelRequired: '请填写模型名',
      referenceRequired: '请选择参考基准',
      referenceAccountRequired: '请选择参考账号',
      baseUrlRequired: '请填写 BaseURL',
      apiKeyRequired: '请填写 API Key',
      accountsLoadFailed: '账号列表加载失败'
    },

    references: {
      sectionTitle: '参考基准',
      sectionDesc: '参考指纹按模型归档，测同一模型的多个账号共用一份参考；官方模型更新会让参考漂移，建议每 1–2 个月重新注册。',
      registerTitle: '注册新参考',
      account: '采样账号',
      accountPlaceholder: '选择可信账号',
      model: '模型名',
      modelPlaceholder: '如 gpt-4o',
      submit: '注册参考',
      submitting: '注册中…',
      registered: '参考注册任务已发起，完成后自动刷新列表',
      registerFailed: '注册参考失败',
      accountRequired: '请选择采样账号',
      modelRequired: '请填写模型名',
      reRegisterStarted: '重新注册任务已发起',
      columns: {
        model: '模型',
        source: '来源',
        enrolledAt: '注册时间',
        cells: 'cell 覆盖',
        actions: '操作'
      },
      sourceAccountSampled: '账号采样',
      sourceZenodo: 'Zenodo 导入',
      stale: '建议重新注册',
      reRegister: '重新注册',
      empty: '暂无参考基准',
      emptyHint: '先注册一个参考基准，或在发起检测时选择「现场注册」',
      loadFailed: '参考基准加载失败',
      cellCount: '{count} 项'
    },

    records: {
      sectionTitle: '检测记录',
      columns: {
        target: '被测对象',
        model: '模型',
        verdict: '可信程度',
        score: 's 值',
        progress: '进度',
        createdAt: '时间',
        actions: '操作'
      },
      empty: '暂无检测记录',
      emptyHint: '在上方发起一次检测',
      loadFailed: '检测记录加载失败',
      detail: '详情',
      accountTarget: '账号 #{id}',
      duration: '耗时 {value}'
    },

    status: {
      running: '进行中',
      done: '已完成',
      failed: '失败'
    },

    // §7.4 四档判定的展示文案
    verdict: {
      consistent: '一致',
      mostlyConsistent: '基本一致',
      warning: '警戒',
      anomalous: '异常',
      insufficient: '证据不足',
      badge: {
        consistent: '✅ 行为特征与声明模型一致',
        mostlyConsistent: '✅ 行为特征与声明模型基本一致',
        warning: '⚠️ 行为特征出现明显偏差，建议复测',
        anomalous: '🔴 行为特征与声明模型显著不符',
        insufficient: '⏳ 样本不足，暂无法判定'
      },
      explain: {
        consistent: '回答问题的统计特征与参考样本吻合。同模型在不同服务商之间的正常波动也会落到这个区间。',
        mostlyConsistent: '回答问题的统计特征与参考样本基本吻合，偏差在同模型跨服务商技术栈的正常波动范围内。',
        warning: '偏差已超出正常波动范围，但尚未达到「不同模型」的典型距离。可能是上游静默更新、量化压缩，也可能是模型被替换。',
        anomalous: '偏差已达到「两个不同模型」的典型距离。统计上所供模型与声明模型不一致（也可能是官方做了重大更新未同步参考基准）。',
        insufficient: '判定需要至少 8 个探测项、每项 10 个有效回答，当前已累积 {k}/{total}。'
      }
    },

    flags: {
      title: '异常标记',
      response_caching: '疑似响应缓存',
      response_cachingDesc: '部分探测项在高温采样下答案分布坍缩且延迟异常低，疑似上游回放缓存答案；对应探测项已从指纹证据中剔除。',
      hidden_reasoning: '疑似隐藏推理，不可审计',
      hidden_reasoningDesc: '一词回答消耗了异常多的输出 token，端点可能存在隐藏推理；单 token 指纹方法不适用，按「不可审计」标记而非误报。',
      not_applicable: '该端点不适用本方法',
      not_applicableDesc: '该端点的输出不是纯模型先验（如强制推理、路由类模型），不做指纹判定。'
    },

    detail: {
      title: '检测报告',
      target: '被测对象',
      model: '模型',
      referenceModel: '参考模型',
      createdAt: '发起时间',
      duration: '耗时',
      score: '距离分 s',
      kn: '判定强度：{k} 个探测项 × 每项约 {n} 个有效样本',
      eerNote: '本判定强度的固有误判率约 9.5%（两类误判合计口径），数据来自论文 165 个模型、2.7 万组对照试验的标定。',
      referenceLine: '参考基准：{source}，注册于 {time}',
      splitHalf: '被测对象自身稳定性：{value}',
      splitHalfNote: '行为本身稳定，不像负载波动，更像模型确实不同。',
      t0Mismatch: 'T=0 快信号：{count} 个探测项的确定性答案与参考不一致（模型被更新或替换的即时提示）',
      cellsTitle: '各探测项分布对比',
      cellColumns: {
        task: '探测项',
        language: '语言',
        jsd: 'JSD',
        valid: '有效样本',
        compare: '被测 top 答案 vs 参考 top 答案',
        note: '备注'
      },
      excluded: {
        response_caching: '疑似缓存，已剔除',
        hidden_reasoning: '隐藏推理，已剔除',
        not_applicable: '不适用，已剔除',
        insufficient_samples: '样本不足，未计入'
      },
      noData: '—',
      running: '任务进行中，完成后自动刷新…',
      failed: '任务失败',
      lastError: '最近一次探测失败：{value}',
      loadFailed: '报告加载失败',
      t0Answers: 'T=0 答案：{value}',
      language: {
        en: '英文',
        zh: '中文'
      },
      tasks: {
        random_number_1_100: '随机数 1–100',
        random_number_1_10: '随机数 1–10',
        favorite_number: '最喜欢的数字',
        random_letter: '随机字母',
        random_color: '随机颜色',
        favorite_color: '最喜欢的颜色',
        random_animal: '随机动物',
        coin_flip: '抛硬币'
      }
    }
  }
}
