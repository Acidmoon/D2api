/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Linework palette: ink, paper, and restrained technical teal
        primary: {
          50: '#eef8f7',
          100: '#d8eeec',
          200: '#b7ddda',
          300: '#88c6c1',
          400: '#52a8a0',
          500: '#2d7f78',
          600: '#236b66',
          700: '#1f5754',
          800: '#1c4543',
          900: '#183836',
          950: '#0d2221'
        },
        // Secondary ink scale
        accent: {
          50: '#f3f6f8',
          100: '#e5ebf0',
          200: '#c9d4dd',
          300: '#a5b7c6',
          400: '#7890a6',
          500: '#4f6a86',
          600: '#3b526d',
          700: '#2d4055',
          800: '#203141',
          900: '#182532',
          950: '#0d151d'
        },
        // Dark linework surfaces
        dark: {
          50: '#f7f8f7',
          100: '#e8eceb',
          200: '#cfd8d5',
          300: '#a8b8b2',
          400: '#78908a',
          500: '#526a64',
          600: '#3d514d',
          700: '#2d3d3a',
          800: '#1b2927',
          900: '#101918',
          950: '#080d0c'
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
        glass: '0 1px 0 rgba(17, 24, 39, 0.08), 0 16px 40px rgba(17, 24, 39, 0.08)',
        'glass-sm': '0 1px 0 rgba(17, 24, 39, 0.08), 0 8px 24px rgba(17, 24, 39, 0.06)',
        glow: '0 0 0 1px rgba(45, 127, 120, 0.18), 0 8px 24px rgba(45, 127, 120, 0.10)',
        'glow-lg': '0 0 0 1px rgba(45, 127, 120, 0.22), 0 18px 44px rgba(45, 127, 120, 0.14)',
        card: '0 1px 0 rgba(17, 24, 39, 0.08), 0 14px 34px rgba(17, 24, 39, 0.06)',
        'card-hover': '0 1px 0 rgba(17, 24, 39, 0.10), 0 18px 44px rgba(17, 24, 39, 0.09)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.65)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, #236b66 0%, #2d4055 100%)',
        'gradient-dark': 'linear-gradient(135deg, #101918 0%, #0d151d 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)',
        'mesh-gradient':
          'linear-gradient(rgba(35, 107, 102, 0.055) 1px, transparent 1px), linear-gradient(90deg, rgba(35, 107, 102, 0.055) 1px, transparent 1px)'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
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
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 20px rgba(20, 184, 166, 0.25)' },
          '100%': { boxShadow: '0 0 30px rgba(20, 184, 166, 0.4)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': '2rem'
      }
    }
  },
  plugins: []
}
