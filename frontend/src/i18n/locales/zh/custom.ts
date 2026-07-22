export default {
  common: {
    apply: '应用',
    clear: '清除',
    creating: '创建中...',
    required: '必填',
    sending: '发送中...',
    skipToContent: '跳到主内容',
    tryAgain: '重试'
  },
  dates: {
    nextMonth: '下月'
  },
  dashboard: {
    activityAnalysis: '活动分析',
    activityHeatmap: '使用活跃',
    heatmapActiveDays: '活跃天数',
    heatmapLess: '少',
    heatmapMore: '多',
    heatmapPeak: '单日峰值',
    heatmapTotal: '本月累计',
    modelRankingTop3: '模型排行榜 Top 3',
    overview: '总览',
    today: '今日',
    totalRequests: '请求次数'
  },
  keys: {
    groupsMustDiffer: '主分组和副分组不能相同',
    primaryGroup: '主分组',
    secondaryGroup: '副分组'
  },
  subscriptionProgress: {
    accountBalance: '账户余额',
    noActiveWallets: '暂无有效订阅额度',
    quotaUnlimited: '订阅额度无限制',
    subscriptionQuota: '订阅额度',
    viewRecharge: '充值/订阅',
    walletHint: '请求会优先消耗订阅额度，订阅额度耗尽后再扣除账户余额。',
    walletTitle: '余额与订阅额度'
  },
  version: {
    dockerUpdateCmd1: 'docker compose pull',
    dockerUpdateCmd2: 'docker compose up -d',
    dockerUpdateGuide:
      '请先确认 docker-compose.yml 中镜像地址为 ghcr.io/acidmoon/d2api:latest，然后执行：',
    dockerUpdateGuideTitle: 'Docker 部署更新指引'
  },
  admin: {
    accounts: {
      fromModel: '源模型',
      toModel: '目标模型',
      messages: {
        accountCreated: '账号创建成功'
      },
      oauth: {
        openai: {
          accessTokenAuth: 'Access Token',
          mobileRefreshTokenAuth: '移动端 Refresh Token'
        }
      }
    },
    announcements: {
      form: {
        subscriptionActive: '有有效订阅',
        subscriptionInactive: '无有效订阅',
        subscriptionStatus: '订阅状态'
      }
    },
    channels: {
      emptyModelsInPricing: '定价中尚未配置模型',
      noGroupsSelected: '未选择分组'
    },
    groups: {
      columns: {
        unavailableAlert: '不可用警告'
      },
      form: {
        unavailableAlert: '不可用警告',
        unavailableAlertHint:
          '开启后，当一次真实请求无法在此分组调度任何账号时，系统会向配置的收件人发送邮件。'
      },
      unavailableAlertDisabled: '未开启',
      unavailableAlertEnabled: '已开启'
    },
    ops: {
      runtime: {
        metricThresholds: '指标阈值',
        metricThresholdsHint: '用于判断运行状态与 SLA 健康度的阈值。',
        requestErrorRateMaxPercent: '请求错误率上限（%）',
        requestErrorRateMaxPercentHint: '请求错误率高于此值时标记为不健康。',
        slaMinPercent: '最低 SLA（%）',
        slaMinPercentHint: 'SLA 低于此值时标记为不健康。',
        ttftP99MaxMs: 'P99 首 Token 延迟上限（毫秒）',
        ttftP99MaxMsHint: 'P99 首 Token 延迟高于此值时标记为不健康。',
        upstreamErrorRateMaxPercent: '上游错误率上限（%）',
        upstreamErrorRateMaxPercentHint: '上游错误率高于此值时标记为不健康。'
      }
    },
    redeem: {
      amountRequired: '请输入有效金额'
    },
    settings: {
      groupUnavailableAlertNotify: {
        addEmail: '添加邮箱',
        description: '当已开启警告的分组在真实请求中无可调度账号时，向这些邮箱发送提醒。',
        emailPlaceholder: '请输入邮箱地址',
        emails: '收件邮箱',
        emailsHint: '只会向已启用的收件人发送邮件；是否告警由各分组的开关控制。',
        title: '分组不可用警告收件人'
      }
    },
    subscriptions: {
      form: {
        dailyLimit: '每日额度（USD）',
        monthlyLimit: '每月额度（USD）',
        planName: '订阅名称',
        weeklyLimit: '每周额度（USD）'
      },
      limitHint: '留空或填 0 表示不限制；请求会优先消耗订阅额度，再扣除普通余额。',
      planHint: '订阅现在是用户级费用额度，不再依赖订阅分组。',
      planNamePlaceholder: '例如 APC 订阅',
      planNameRequired: '请输入订阅名称'
    },
    users: {
      passwordCopied: '密码已复制'
    }
  }
}
