/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./web/templates/**/*.html",
    "./web/static/js/**/*.js",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // =============================================
        // CSS Variable-based colors (for theming)
        // Use these in new components
        // =============================================
        'primary': {
          DEFAULT: 'var(--color-primary)',
          hover: 'var(--color-primary-hover)',
          active: 'var(--color-primary-active)',
          subtle: 'var(--color-primary-subtle)',
        },
        'secondary': {
          DEFAULT: 'var(--color-secondary)',
          hover: 'var(--color-secondary-hover)',
          subtle: 'var(--color-secondary-subtle)',
        },
        'accent': {
          DEFAULT: 'var(--color-accent)',
          hover: 'var(--color-accent-hover)',
          text: 'var(--color-accent-text)',
        },
        'danger': {
          DEFAULT: 'var(--color-danger)',
          hover: 'var(--color-danger-hover)',
          subtle: 'var(--color-danger-subtle)',
        },
        'success': {
          DEFAULT: 'var(--color-success)',
          hover: 'var(--color-success-hover)',
          subtle: 'var(--color-success-subtle)',
        },
        'warning': {
          DEFAULT: 'var(--color-warning)',
          hover: 'var(--color-warning-hover)',
          subtle: 'var(--color-warning-subtle)',
        },
        'info': {
          DEFAULT: 'var(--color-info)',
          hover: 'var(--color-info-hover)',
          subtle: 'var(--color-info-subtle)',
        },
        'surface': {
          DEFAULT: 'var(--color-surface)',
          raised: 'var(--color-surface-raised)',
          overlay: 'var(--color-surface-overlay)',
          inset: 'var(--color-surface-inset)',
        },
        'background': 'var(--color-background)',
        'border': {
          DEFAULT: 'var(--color-border)',
          strong: 'var(--color-border-strong)',
          hover: 'var(--color-border-hover)',
        },

        // =============================================
        // Legacy brand colors (keep for backward compatibility)
        // These use hardcoded values
        // =============================================
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
          800: '#1A4D2E',
          900: '#0D2617',
        },
        'gold': {
          DEFAULT: '#FCD116',
          50: '#FFFBEB',
          100: '#FEF3C7',
          200: '#FDE68A',
          300: '#FCD116',
          400: '#FACC15',
          500: '#EAB308',
          600: '#CA8A04',
          700: '#A16207',
          800: '#854D0E',
          900: '#713F12',
        },
        'clay': {
          DEFAULT: '#8B7355',
          50: '#F7F5F3',
          100: '#EFEAE5',
          200: '#DDD4C9',
          300: '#CBBEAD',
          400: '#B9A891',
          500: '#A79275',
          600: '#8B7355',
          700: '#6B5842',
          800: '#4B3E2E',
          900: '#2B241B',
        },
        'cream': '#E8E4DF',
        'alert': '#CE1126',
      },
      fontFamily: {
        sans: ['var(--font-family-sans)', 'system-ui', 'sans-serif'],
        mono: ['var(--font-family-mono)', 'monospace'],
      },
      borderRadius: {
        'sm': 'var(--radius-sm)',
        'DEFAULT': 'var(--radius-DEFAULT)',
        'md': 'var(--radius-md)',
        'lg': 'var(--radius-lg)',
        'xl': 'var(--radius-xl)',
        '2xl': 'var(--radius-2xl)',
        '3xl': 'var(--radius-3xl)',
      },
      boxShadow: {
        'xs': 'var(--shadow-xs)',
        'sm': 'var(--shadow-sm)',
        'DEFAULT': 'var(--shadow-DEFAULT)',
        'md': 'var(--shadow-md)',
        'lg': 'var(--shadow-lg)',
        'xl': 'var(--shadow-xl)',
        '2xl': 'var(--shadow-2xl)',
      },
      zIndex: {
        'dropdown': 'var(--z-dropdown)',
        'sticky': 'var(--z-sticky)',
        'fixed': 'var(--z-fixed)',
        'modal-backdrop': 'var(--z-modal-backdrop)',
        'modal': 'var(--z-modal)',
        'popover': 'var(--z-popover)',
        'tooltip': 'var(--z-tooltip)',
      },
      transitionDuration: {
        '75': 'var(--duration-75)',
        '100': 'var(--duration-100)',
        '150': 'var(--duration-150)',
        '200': 'var(--duration-200)',
        '300': 'var(--duration-300)',
        '500': 'var(--duration-500)',
      },
      textColor: {
        'DEFAULT': 'var(--color-text)',
        'secondary': 'var(--color-text-secondary)',
        'tertiary': 'var(--color-text-tertiary)',
        'inverse': 'var(--color-text-inverse)',
        'on-primary': 'var(--color-text-on-primary)',
        'on-accent': 'var(--color-text-on-accent)',
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
