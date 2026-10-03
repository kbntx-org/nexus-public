export default {
  multipass: true,
  plugins: [
    {
      name: 'preset-default',
      params: {
        overrides: {
          inlineStyles: false,
          minifyStyles: false,
          cleanupIds: false,
          removeUnknownsAndDefaults: { keepRoleAttr: true }
        }
      }
    }
  ]
};
