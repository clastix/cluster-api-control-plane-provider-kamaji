// commitlint.config.js
export default {
    extends: ['@commitlint/config-conventional'],
    ignores: [(message) => /^Bumps \[.+]\(.+\) from .+ to .+\.$/m.test(message)],
};
