import animate from 'tailwindcss-animate'

/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
  	extend: {
  		colors: {
  			primary: {
  				'50': '#eef8f6',
  				'100': '#d9efeb',
  				'200': '#b7dfd8',
  				'300': '#86c9bf',
  				'400': '#52afa3',
  				'500': '#2b968c',
  				'600': '#0f766e',
  				'700': '#0b5d57',
  				'800': '#0a4a46',
  				'900': '#083f3c',
  				'950': '#032624',
  				DEFAULT: 'hsl(var(--primary))',
  				foreground: 'hsl(var(--primary-foreground))'
  			},
  			accent: {
  				'50': '#f7f7f4',
  				'100': '#f0f0ec',
  				'200': '#ddddd6',
  				'300': '#cfcfc8',
  				'400': '#a6a69c',
  				'500': '#77776f',
  				'600': '#4b4b45',
  				'700': '#343430',
  				'800': '#20201e',
  				'900': '#171717',
  				'950': '#0b0b0b',
  				DEFAULT: 'hsl(var(--accent))',
  				foreground: 'hsl(var(--accent-foreground))'
  			},
  			dark: {
  				'50': '#f4f4ef',
  				'100': '#e8e8de',
  				'200': '#c7c7bd',
  				'300': '#a6a69c',
  				'400': '#85857b',
  				'500': '#66665f',
  				'600': '#4a4a45',
  				'700': '#34342f',
  				'800': '#20201e',
  				'900': '#171717',
  				'950': '#111111'
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

