import { defineConfig } from 'vite';
import preact from '@preact/preset-vite';
import { resolve } from 'path';

export default defineConfig({
  plugins: [preact()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  build: {
    lib: {
      entry: resolve(__dirname, 'src/index.ts'),
      name: 'Rekko',
      fileName: (format) => `rekko.${format === 'umd' ? 'min' : format}.js`,
      formats: ['umd', 'es'],
    },
    rollupOptions: {
      output: {
        assetFileNames: 'rekko.[ext]',
        globals: {},
      },
    },
    minify: 'esbuild',
    sourcemap: true,
    target: 'es2020',
  },
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
});
