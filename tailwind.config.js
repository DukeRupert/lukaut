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
        // Brand colors - Slate Navy + Safety Orange
        // =============================================
        'navy': {
          DEFAULT: '#1E3A5F',
          50: '#F0F4F8',
          100: '#D9E2EC',
          200: '#BCCCDC',
          300: '#9FB3C8',
          400: '#7B93AD',
          500: '#5A7491',
          600: '#3D5A80',
          700: '#2E4A6B',
          800: '#1E3A5F',
          900: '#102A43',
          950: '#0A1929',
        },
        'safety-orange': {
          DEFAULT: '#FF6B35',
          50: '#FFF4F0',
          100: '#FFE4D9',
          200: '#FFC9B3',
          300: '#FFAD8C',
          400: '#FF8C5A',
          500: '#FF6B35',
          600: '#E85A2A',
          700: '#C44A22',
          800: '#9F3B1B',
          900: '#7A2D14',
          950: '#4A1B0C',
        },
        'alert': '#DC2626',
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
