import animate from 'tailwindcss-animate'

/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
  	extend: {
  		colors: {
  			primary: {
  				'50': '#f0efff',
  				'100': '#e3e1ff',
  				'200': '#c9c6ff',
  				'300': '#a8a4ff',
  				'400': '#8682ff',
  				'500': '#6c68ff',
  				'600': '#5b58ff',
  				'700': '#4643e8',
  				'800': '#3432b4',
  				'900': '#26248a',
  				'950': '#171660',
  				DEFAULT: 'hsl(var(--primary))',
  				foreground: 'hsl(var(--primary-foreground))'
  			},
  			accent: {
  				'50': '#f7f8fa',
  				'100': '#f2f4f8',
  				'200': '#e6e9ef',
  				'300': '#d5dae2',
  				'400': '#a9b2c0',
  				'500': '#7a8290',
  				'600': '#4e5969',
  				'700': '#363e4a',
  				'800': '#1d2129',
  				'900': '#12141a',
  				'950': '#0b0c0f',
  				DEFAULT: 'hsl(var(--accent))',
  				foreground: 'hsl(var(--accent-foreground))'
  			},
  			dark: {
  				'50': '#f2f4f8',
  				'100': '#e6e9ef',
  				'200': '#c5cbd5',
  				'300': '#a9b2c0',
  				'400': '#7a8290',
  				'500': '#4e5969',
  				'600': '#363e4a',
  				'700': '#2a2e37',
  				'800': '#1d2026',
  				'900': '#15171c',
  				'950': '#0b0c0f'
  			},
  			background: 'hsl(var(--background))',
  			foreground: 'hsl(var(--foreground))',
  			card: {
  				DEFAULT: 'hsl(var(--card))',
  				foreground: 'hsl(var(--card-foreground))'
  			},
  			popover: {
  				DEFAULT: 'hsl(var(--popover))',
  				foreground: 'hsl(var(--popover-foreground))'
  			},
  			secondary: {
  				DEFAULT: 'hsl(var(--secondary))',
  				foreground: 'hsl(var(--secondary-foreground))'
  			},
  			muted: {
  				DEFAULT: 'hsl(var(--muted))',
  				foreground: 'hsl(var(--muted-foreground))'
  			},
  			destructive: {
  				DEFAULT: 'hsl(var(--destructive))',
  				foreground: 'hsl(var(--destructive-foreground))'
  			},
  			border: 'hsl(var(--border))',
  			input: 'hsl(var(--input))',
  			ring: 'hsl(var(--ring))',
  			brand: {
  				DEFAULT: 'hsl(var(--brand))',
  				foreground: 'hsl(var(--brand-foreground))'
  			},
  			chart: {
  				'1': 'hsl(var(--chart-1))',
  				'2': 'hsl(var(--chart-2))',
  				'3': 'hsl(var(--chart-3))',
  				'4': 'hsl(var(--chart-4))',
  				'5': 'hsl(var(--chart-5))'
  			}
  		},
  		fontFamily: {
  			sans: [
  				'Inter',
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
  			mono: [
  				'ui-monospace',
  				'SFMono-Regular',
  				'Menlo',
  				'Monaco',
  				'Consolas',
  				'monospace'
  			]
  		},
  		boxShadow: {
  			'nm-sm': 'var(--nm-shadow-raised-sm)',
  			nm: 'var(--nm-shadow-raised)',
  			'nm-lg': 'var(--nm-shadow-raised-lg)',
  			'nm-inset': 'var(--nm-shadow-inset)',
  			'nm-pressed': 'var(--nm-shadow-pressed)',
  			card: 'var(--nm-shadow-raised-sm)',
  			'card-hover': 'var(--nm-shadow-raised-lg)',
  			glass: 'var(--nm-shadow-raised)'
  		},
  		borderRadius: {
  			'4xl': '1.5rem',
  			'5xl': '2rem',
  			lg: 'var(--radius)',
  			md: 'calc(var(--radius) - 2px)',
  			sm: 'calc(var(--radius) - 4px)'
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
  				'0%': {
  					opacity: '0'
  				},
  				'100%': {
  					opacity: '1'
  				}
  			},
  			slideUp: {
  				'0%': {
  					opacity: '0',
  					transform: 'translateY(10px)'
  				},
  				'100%': {
  					opacity: '1',
  					transform: 'translateY(0)'
  				}
  			},
  			slideDown: {
  				'0%': {
  					opacity: '0',
  					transform: 'translateY(-10px)'
  				},
  				'100%': {
  					opacity: '1',
  					transform: 'translateY(0)'
  				}
  			},
  			slideInRight: {
  				'0%': {
  					opacity: '0',
  					transform: 'translateX(20px)'
  				},
  				'100%': {
  					opacity: '1',
  					transform: 'translateX(0)'
  				}
  			},
  			scaleIn: {
  				'0%': {
  					opacity: '0',
  					transform: 'scale(0.95)'
  				},
  				'100%': {
  					opacity: '1',
  					transform: 'scale(1)'
  				}
  			}
  		}
  	}
  },
  plugins: [animate]
}

