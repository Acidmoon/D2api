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
        // Neumorphic raised (light top-left / dark bottom-right)
        'nm-sm': '6px 6px 12px rgba(166, 173, 189, 0.45), -4px -4px 8px rgba(255, 255, 255, 0.85)',
        nm: '8px 8px 16px rgba(166, 173, 189, 0.50), -6px -6px 12px rgba(255, 255, 255, 0.90)',
        'nm-lg': '12px 12px 24px rgba(166, 173, 189, 0.55), -8px -8px 16px rgba(255, 255, 255, 0.95)',
        // Neumorphic inset (pressed / sunken)
        'nm-inset-sm': 'inset 3px 3px 6px rgba(166, 173, 189, 0.40), inset -2px -2px 4px rgba(255, 255, 255, 0.80)',
        'nm-inset': 'inset 5px 5px 10px rgba(166, 173, 189, 0.45), inset -3px -3px 6px rgba(255, 255, 255, 0.85)',
        'nm-inset-lg': 'inset 8px 8px 16px rgba(166, 173, 189, 0.50), inset -4px -4px 8px rgba(255, 255, 255, 0.90)',
        // Neumorphic dark mode
        'nm-dark-sm': '6px 6px 12px rgba(15, 16, 20, 0.65), -4px -4px 8px rgba(55, 58, 68, 0.35)',
        'nm-dark': '8px 8px 16px rgba(15, 16, 20, 0.70), -6px -6px 12px rgba(55, 58, 68, 0.40)',
        'nm-dark-lg': '12px 12px 24px rgba(15, 16, 20, 0.75), -8px -8px 16px rgba(55, 58, 68, 0.45)',
        'nm-dark-inset': 'inset 5px 5px 10px rgba(15, 16, 20, 0.55), inset -3px -3px 6px rgba(55, 58, 68, 0.25)',
        'nm-dark-inset-lg': 'inset 8px 8px 16px rgba(15, 16, 20, 0.60), inset -4px -4px 8px rgba(55, 58, 68, 0.30)'
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
