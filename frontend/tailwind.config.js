/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Neumorphism soft accent — muted lavender-blue
        primary: {
          50: '#f4f5f9',
          100: '#e7e9f2',
          200: '#cfd3e5',
          300: '#aeb5cf',
          400: '#8890b5',
          500: '#6b75a0',
          600: '#576089',
          700: '#48506f',
          800: '#3d445d',
          900: '#343a4f',
          950: '#1f2332'
        },
        // Neumorphism warm accent — soft rose-grey
        accent: {
          50: '#f8f6fa',
          100: '#efecf3',
          200: '#e0d9e8',
          300: '#c8bdd6',
          400: '#a997bd',
          500: '#8e76a5',
          600: '#745d8b',
          700: '#5f4b72',
          800: '#4f3f5f',
          900: '#433650',
          950: '#261d2f'
        },
        // Neumorphic dark surfaces
        dark: {
          50: '#f5f5f6',
          100: '#e7e7e9',
          200: '#d1d1d5',
          300: '#b0b0b7',
          400: '#888893',
          500: '#6d6d78',
          600: '#5a5a65',
          700: '#484852',
          800: '#34343d',
          900: '#23232a',
          950: '#16161b'
        }
      },
      fontFamily: {
        sans: [
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        // Neumorphic shadows are driven by CSS variables (defined in style.css),
        // so light/dark switch automatically — no separate -dark variants needed.
        'nm-sm': 'var(--nm-shadow-raised-sm)',
        nm: 'var(--nm-shadow-raised)',
        'nm-lg': 'var(--nm-shadow-raised-lg)',
        'nm-inset': 'var(--nm-shadow-inset)',
        'nm-pressed': 'var(--nm-shadow-pressed)',
        // Semantic aliases consumed by .card / .glass-card / .card-hover and
        // any inline `shadow-card`/`shadow-glass` usages.
        card: 'var(--nm-shadow-raised-sm)',
        'card-hover': 'var(--nm-shadow-raised-lg)',
        glass: 'var(--nm-shadow-raised)'
      },
      borderRadius: {
        '4xl': '1.5rem',
        '5xl': '2rem'
      },
      animation: {
        'fade-in': 'fadeIn 0.25s ease-out',
        'slide-up': 'slideUp 0.25s ease-out',
        'slide-down': 'slideDown 0.25s ease-out',
        'slide-in-right': 'slideInRight 0.25s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        }
      }
    }
  },
  plugins: []
}
