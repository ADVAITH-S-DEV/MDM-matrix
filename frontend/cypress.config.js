import { defineConfig } from 'cypress';

export default defineConfig({
  allowCypressEnv: false,
  e2e: {
    supportFile: false,
    baseUrl: 'http://127.0.0.1:5173',
    video: false,
    screenshotOnRunFailure: true,
  },
});
