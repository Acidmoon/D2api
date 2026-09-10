export default {
  imageStudio: {
    title: 'Image Studio',
    description: 'Generate images with your own API key, then preview them full size or download the originals.',
    key: {
      label: 'Creation group API key',
      placeholder: 'Select an API key',
      hint: 'Group: {group}. Requests are billed to this key.',
      empty: 'No API key under the creation group. Create one under API Keys first.',
      refresh: 'Refresh',
      create: 'Create API key'
    },
    form: {
      model: 'Model',
      modelPlaceholder: 'Select a model',
      modelsLoading: 'Loading models...',
      modelsEmpty: 'This group has no available models yet.',
      prompt: 'Prompt',
      promptPlaceholder: 'Describe the image you want...',
      size: 'Size',
      sizeAuto: 'Auto',
      count: 'Count',
      generate: 'Generate',
      generating: 'Generating...',
      costHint: 'Each generated image is billed through the selected group.'
    },
    result: {
      title: 'Results',
      clear: 'Clear',
      emptyTitle: 'No images yet',
      emptyDescription: 'Write a prompt on the left and click Generate.',
      noKeyTitle: 'No creation group API key',
      noKeyDescription: 'Create an API key under the creation group to generate images here.',
      loading: 'Generating, this can take a while...',
      open: 'View',
      download: 'Download',
      downloading: 'Downloading...',
      downloadOriginal: 'Download original',
      revisedPrompt: 'Revised prompt',
      generated: 'Generated {count} image(s)'
    },
    lightbox: {
      title: 'Preview'
    },
    history: {
      open: 'History',
      title: 'Generation history',
      detailTitle: 'History detail',
      emptyTitle: 'No history yet',
      emptyDescription: 'Generated images are kept for 7 days and only visible to you.',
      prompt: 'Prompt',
      revisedPrompt: 'Revised prompt',
      expiresAt: 'Retained until ',
      imageCount: '{count} image(s)',
      view: 'View',
      back: 'Back to list',
      delete: 'Delete',
      deleted: 'History deleted',
      deleteFailed: 'Failed to delete: {message}',
      saveFailed: 'Failed to save history: {message}',
      prev: 'Previous',
      next: 'Next',
      pageInfo: 'Page {page} / {total}'
    },
    errors: {
      keyRequired: 'Select an API key first.',
      modelRequired: 'Select a model first.',
      promptRequired: 'Enter a prompt first.',
      modelsFailed: 'Failed to load models: {message}',
      generateFailed: 'Generation failed: {message}',
      downloadFailed: 'Download failed: {message}',
      emptyResult: 'The upstream returned no image.'
    }
  }
}
