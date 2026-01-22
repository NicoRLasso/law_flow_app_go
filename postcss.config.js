export default {
  plugins: {
    "@tailwindcss/postcss": {},
    ...(process.env.NODE_ENV === 'production' ? {
      "cssnano": {
        "preset": "default",
      },
    } : {}),
  },
  watchOptions: {
    ignored: [
      '**/node_modules/**',
      '**/*_templ.go',
      '**/static/css/bundle.min.css',
      '**/bin/**',
      '**/.git/**'
    ]
  }
}
