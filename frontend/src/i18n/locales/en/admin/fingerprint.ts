export default {
  fingerprint: {
    title: 'Model Fingerprint Audit',
    description: 'Verify model identity for system accounts or external endpoints by comparing single-word answer distributions against a reference fingerprint',
    // §7.4 one-sentence explanation of the method
    help: 'How it works: we ask the upstream one-word questions like "Name a random number between 1 and 100" and sample repeatedly. Every model has its own stable verbal tic (preference distribution) for such questions, and different models differ a lot. The score s measures how far the target’s tic is from the genuine model’s: 0 means identical, larger means more suspicious.',

    create: {
      sectionTitle: 'Start an Audit',
      tabAccount: 'Audit an Account',
      tabExternal: 'Audit an External Endpoint',
      account: 'Target account',
      accountPlaceholder: 'Select an account in the system',
      model: 'Model name',
      modelPlaceholder: 'The model the target claims to serve, e.g. gpt-4o',
      referenceMode: 'Reference',
      referenceModeExisting: 'Use existing reference',
      referenceModeEnroll: 'Enroll on the spot from a trusted account',
      referenceExisting: 'Existing reference',
      referenceExistingPlaceholder: 'Select an enrolled reference',
      referenceAccount: 'Reference account',
      referenceAccountPlaceholder: 'Select a trusted account (enroll first, then audit)',
      referenceEnrollHint: 'A reference for this model will be enrolled from the reference account first, then the target is audited',
      baseUrl: 'Base URL',
      baseUrlPlaceholder: 'https://api.example.com',
      apiKey: 'API Key',
      apiKeyPlaceholder: 'Used only for this audit',
      apiKeyHint: 'The API key is held only while the task runs; it is never written to any file or log',
      provider: 'Provider',
      apiMode: 'API mode',
      keepRaw: 'Keep raw samples',
      keepRawHint: 'Attach the raw answer of every probe to the report for manual review (the report file gets much larger)',
      submit: 'Start Audit',
      submitting: 'Starting…',
      created: 'Audit task started; progress is shown in the records below',
      createFailed: 'Failed to start the audit',
      accountRequired: 'Please select a target account',
      modelRequired: 'Please enter the model name',
      referenceRequired: 'Please select a reference',
      referenceAccountRequired: 'Please select a reference account',
      baseUrlRequired: 'Please enter the base URL',
      apiKeyRequired: 'Please enter the API key',
      accountsLoadFailed: 'Failed to load accounts'
    },

    references: {
      sectionTitle: 'References',
      sectionDesc: 'Reference fingerprints are archived by model and shared by all accounts claiming the same model. Official model updates make references drift, so re-enroll every 1–2 months.',
      registerTitle: 'Enroll a New Reference',
      account: 'Sampling account',
      accountPlaceholder: 'Select a trusted account',
      model: 'Model name',
      modelPlaceholder: 'e.g. gpt-4o',
      submit: 'Enroll Reference',
      submitting: 'Enrolling…',
      registered: 'Reference enrollment started; the list refreshes when it finishes',
      registerFailed: 'Failed to enroll the reference',
      accountRequired: 'Please select a sampling account',
      modelRequired: 'Please enter the model name',
      reRegisterStarted: 'Re-enrollment task started',
      columns: {
        model: 'Model',
        source: 'Source',
        enrolledAt: 'Enrolled At',
        cells: 'Cell Coverage',
        actions: 'Actions'
      },
      sourceAccountSampled: 'Account sampled',
      sourceZenodo: 'Zenodo import',
      stale: 'Re-enrollment recommended',
      reRegister: 'Re-enroll',
      empty: 'No references yet',
      emptyHint: 'Enroll a reference first, or choose "enroll on the spot" when starting an audit',
      loadFailed: 'Failed to load references',
      cellCount: '{count} cells'
    },

    records: {
      sectionTitle: 'Audit Records',
      columns: {
        target: 'Target',
        model: 'Model',
        verdict: 'Verdict',
        score: 's',
        progress: 'Progress',
        createdAt: 'Time',
        actions: 'Actions'
      },
      empty: 'No audit records yet',
      emptyHint: 'Start an audit above',
      loadFailed: 'Failed to load audit records',
      detail: 'Details',
      accountTarget: 'Account #{id}',
      duration: 'Duration {value}'
    },

    status: {
      running: 'Running',
      done: 'Done',
      failed: 'Failed'
    },

    // §7.4 display wording for the four verdict bands
    verdict: {
      consistent: 'Consistent',
      mostlyConsistent: 'Mostly Consistent',
      warning: 'Warning',
      anomalous: 'Anomalous',
      insufficient: 'Insufficient',
      badge: {
        consistent: '✅ Behavioral fingerprint matches the claimed model',
        mostlyConsistent: '✅ Behavioral fingerprint basically matches the claimed model',
        warning: '⚠️ Behavioral fingerprint deviates noticeably; re-test recommended',
        anomalous: '🔴 Behavioral fingerprint clearly does not match the claimed model',
        insufficient: '⏳ Not enough samples to decide'
      },
      explain: {
        consistent: 'The statistical traits of the answers match the reference samples. Normal variation of the same model across different providers also falls in this range.',
        mostlyConsistent: 'The statistical traits of the answers basically match the reference samples; the deviation is within the normal range of the same model served by different provider stacks.',
        warning: 'The deviation exceeds normal variation but has not reached the typical distance of a different model. It may be a silent upstream update, quantization, or a model swap.',
        anomalous: 'The deviation has reached the typical distance of two different models. Statistically, the served model does not match the claimed one (a major official update without re-enrolling the reference is also possible).',
        insufficient: 'A verdict needs at least 8 probe cells with 10 valid answers each; currently accumulated {k}/{total}.'
      }
    },

    flags: {
      title: 'Anomaly Flags',
      response_caching: 'Suspected response caching',
      response_cachingDesc: 'Some cells collapsed to a single answer under high-temperature sampling with abnormally low latency, suggesting replayed cached answers; the affected cells were excluded from the fingerprint evidence.',
      hidden_reasoning: 'Suspected hidden reasoning; not auditable',
      hidden_reasoningDesc: 'Single-word answers consumed an abnormally high number of output tokens, suggesting hidden reasoning; the single-token fingerprint method does not apply, so this is marked as not auditable rather than a mismatch.',
      not_applicable: 'Method not applicable to this endpoint',
      not_applicableDesc: 'The output of this endpoint is not a pure model prior (e.g. forced reasoning or router-type models), so no fingerprint verdict is made.'
    },

    detail: {
      title: 'Audit Report',
      target: 'Target',
      model: 'Model',
      referenceModel: 'Reference Model',
      createdAt: 'Started At',
      duration: 'Duration',
      score: 'Distance score s',
      kn: 'Verdict strength: {k} probe cells × about {n} valid samples per cell',
      eerNote: 'The inherent error rate at this verdict strength is about 9.5% (both error kinds combined), calibrated on 165 models and 27,000 controlled comparisons in the paper.',
      referenceLine: 'Reference: {source}, enrolled at {time}',
      splitHalf: 'Target self-stability: {value}',
      splitHalfNote: 'The behavior is stable on its own; this looks less like load fluctuation and more like a genuinely different model.',
      t0Mismatch: 'T=0 fast signal: {count} cells have deterministic answers that differ from the reference (an immediate hint that the model was updated or swapped)',
      cellsTitle: 'Per-cell Distribution Comparison',
      cellColumns: {
        task: 'Probe',
        language: 'Language',
        jsd: 'JSD',
        valid: 'Valid Samples',
        compare: 'Target top answers vs reference top answers',
        note: 'Note'
      },
      excluded: {
        response_caching: 'Suspected caching, excluded',
        hidden_reasoning: 'Hidden reasoning, excluded',
        not_applicable: 'Not applicable, excluded',
        insufficient_samples: 'Too few samples, not counted'
      },
      noData: '—',
      running: 'Task is running; this view refreshes automatically when it finishes…',
      failed: 'Task failed',
      loadFailed: 'Failed to load the report',
      t0Answers: 'T=0 answers: {value}',
      language: {
        en: 'English',
        zh: 'Chinese'
      },
      tasks: {
        random_number_1_100: 'Random number 1–100',
        random_number_1_10: 'Random number 1–10',
        favorite_number: 'Favorite number',
        random_letter: 'Random letter',
        random_color: 'Random color',
        favorite_color: 'Favorite color',
        random_animal: 'Random animal',
        coin_flip: 'Coin flip'
      }
    }
  }
}
