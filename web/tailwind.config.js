/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
  colors: {
    background: {
      light: '#f3f4f6',
      dark: '#1a202c',
    },
    surface: {
      light: '#ffffff',
      dark: '#2d3748',
    },
    text: {
      light: '#111827',
      dark: '#f7fafc',
    },
  },
  fontFamily: {
    sans: ['Inter', 'sans-serif'],
  },
},
  },
  plugins: [],
}
