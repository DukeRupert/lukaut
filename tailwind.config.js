/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./web/templates/**/*.html",
    "./web/static/js/**/*.js",
  ],
  theme: {
    extend: {
      colors: {
        // Primary brand color - PNG highlands rainforest
        'forest': {
          DEFAULT: '#1A4D2E',
          50: '#E8F5EC',
          100: '#C5E6CE',
          200: '#9FD4AF',
          300: '#79C290',
          400: '#53B071',
          500: '#2D9E52',
          600: '#267F43',
          700: '#1F6134',
          800: '#1A4D2E',  // DEFAULT
          900: '#0D2617',
        },
        // Accent color - Bird-of-Paradise, PNG flag
        'gold': {
          DEFAULT: '#FCD116',
          50: '#FFFBEB',
          100: '#FEF3C7',
          200: '#FDE68A',
          300: '#FCD116',  // DEFAULT
          400: '#FACC15',
          500: '#EAB308',
          600: '#CA8A04',
          700: '#A16207',
          800: '#854D0E',
          900: '#713F12',
        },
        // Secondary text - Sepik River clay
        'clay': {
          DEFAULT: '#8B7355',
          50: '#F7F5F3',
          100: '#EFEAE5',
          200: '#DDD4C9',
          300: '#CBBEAD',
          400: '#B9A891',
          500: '#A79275',
          600: '#8B7355',  // DEFAULT
          700: '#6B5842',
          800: '#4B3E2E',
          900: '#2B241B',
        },
        // Background - Traditional tapa cloth
        'cream': '#E8E4DF',
        // Alert/Error - PNG flag red
        'alert': '#CE1126',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
        mono: ['JetBrains Mono', 'Menlo', 'Monaco', 'Consolas', 'monospace'],
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
