export default {
  common: {
    apply: 'Apply',
    clear: 'Clear',
    creating: 'Creating...',
    required: 'Required',
    sending: 'Sending...',
    skipToContent: 'Skip to main content',
    tryAgain: 'Try again'
  },
  dates: {
    nextMonth: 'Next Month'
  },
  dashboard: {
    activityAnalysis: 'Activity',
    activityHeatmap: 'Activity Heatmap',
    heatmapActiveDays: 'Active days',
    heatmapLess: 'Less',
    heatmapMore: 'More',
    heatmapPeak: 'Daily peak',
    heatmapTotal: 'Month total',
    modelRankingTop3: 'Top 3 Models',
    overview: 'Overview',
    today: 'Today',
    totalRequests: 'Requests'
  },
  keys: {
    groupsMustDiffer: 'Primary and secondary groups cannot be the same',
    primaryGroup: 'Primary Group',
    secondaryGroup: 'Secondary Group'
  },
  subscriptionProgress: {
    accountBalance: 'Account balance',
    noActiveWallets: 'No active subscription quota',
    quotaUnlimited: 'Unlimited subscription quota',
    subscriptionQuota: 'Subscription quota',
    viewRecharge: 'Recharge / subscribe',
    walletHint: 'Requests consume subscription quota first, then fall back to account balance.',
    walletTitle: 'Balance and subscription quota'
  },
  version: {
    dockerUpdateCmd1: 'docker compose pull',
    dockerUpdateCmd2: 'docker compose up -d',
    dockerUpdateGuide:
      'Confirm that docker-compose.yml uses ghcr.io/acidmoon/d2api:latest, then run:',
    dockerUpdateGuideTitle: 'Docker Deployment Update Guide'
  },
  admin: {
    accounts: {
      fromModel: 'Source model',
      toModel: 'Target model',
      messages: {
        accountCreated: 'Account created successfully'
      },
      oauth: {
        openai: {
          accessTokenAuth: 'Access token',
          mobileRefreshTokenAuth: 'Mobile refresh token'
        }
      }
    },
    announcements: {
      form: {
        subscriptionActive: 'Has active subscription',
        subscriptionInactive: 'No active subscription',
        subscriptionStatus: 'Subscription status'
      }
    },
    channels: {
      emptyModelsInPricing: 'No models are configured for pricing',
      noGroupsSelected: 'No groups selected'
    },
    groups: {
      columns: {
        unavailableAlert: 'Unavailable Alert'
      },
      form: {
        unavailableAlert: 'Unavailable Alert',
        unavailableAlertHint:
          'When enabled, the system emails recipients if a real request cannot schedule any account in this group.'
      },
      unavailableAlertDisabled: 'Disabled',
      unavailableAlertEnabled: 'Enabled'
    },
    ops: {
      runtime: {
        metricThresholds: 'Metric thresholds',
        metricThresholdsHint: 'Thresholds used to evaluate runtime health and SLA status.',
        requestErrorRateMaxPercent: 'Maximum request error rate (%)',
        requestErrorRateMaxPercentHint: 'Mark the runtime unhealthy above this request error rate.',
        slaMinPercent: 'Minimum SLA (%)',
        slaMinPercentHint: 'Mark the runtime unhealthy below this SLA.',
        ttftP99MaxMs: 'Maximum P99 TTFT (ms)',
        ttftP99MaxMsHint: 'Mark the runtime unhealthy above this P99 time to first token.',
        upstreamErrorRateMaxPercent: 'Maximum upstream error rate (%)',
        upstreamErrorRateMaxPercentHint:
          'Mark the runtime unhealthy above this upstream error rate.'
      }
    },
    redeem: {
      amountRequired: 'Please enter a valid amount'
    },
    settings: {
      groupUnavailableAlertNotify: {
        addEmail: 'Add Email',
        description:
          'Send alerts to these recipients when an enabled group cannot schedule any account for a real request.',
        emailPlaceholder: 'Enter email address',
        emails: 'Recipient Emails',
        emailsHint:
          'Only enabled recipients receive email; each group controls whether unavailable alerts are active.',
        title: 'Group Unavailable Alert Recipients'
      }
    },
    subscriptions: {
      form: {
        dailyLimit: 'Daily Quota (USD)',
        monthlyLimit: 'Monthly Quota (USD)',
        planName: 'Subscription Name',
        weeklyLimit: 'Weekly Quota (USD)'
      },
      limitHint:
        'Leave blank or enter 0 for no limit. Requests consume subscription quota before regular balance.',
      planHint:
        'Subscriptions are user-level billing quotas and no longer depend on subscription groups.',
      planNamePlaceholder: 'For example, APC Subscription',
      planNameRequired: 'Please enter a subscription name'
    },
    users: {
      passwordCopied: 'Password copied'
    }
  }
}
