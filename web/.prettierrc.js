module.exports = {
  printWidth: 140,
  useTabs: false,
  semi: true,
  singleQuote: false,
  endOfLine: "auto",
  plugins: [require.resolve("@trivago/prettier-plugin-sort-imports")],
  importOrder: ["<BUILTIN_MODULES>", "<THIRD_PARTY_MODULES>", "^@/((?!css).+)", "^[./]", "^(.+).css"],
};
