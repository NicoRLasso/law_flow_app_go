export default {
  plugins: {
    "@tailwindcss/postcss": {},
  },
  watchOptions: {
    ignored: [
      '**/node_modules/**',
      '**/*_templ.go',
      '**/static/css/output.css',
      '**/bin/**',
      '**/.git/**'
    ]
  }
}
