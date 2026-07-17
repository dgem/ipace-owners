export default {
  extends: ['stylelint-config-recommended'],
  ignoreFiles: ['_site/**'],
  rules: {
    'selector-class-pattern': null,
    'no-duplicate-selectors': null,
    'no-descending-specificity': null,
    'property-no-deprecated': null,
  },
};
