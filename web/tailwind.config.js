/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#f0fdf4',
          500: '#22c55e',
          600: '#16a34a',
          900: '#14532d',
          950: '#052e16',
        },
        surface: {
          800: '#1e222b',
          850: '#171a22',
          900: '#0f1117',
          950: '#0a0c10',
        }
      }
    },
  },
  plugins: [],
}
