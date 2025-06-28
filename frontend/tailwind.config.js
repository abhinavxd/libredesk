const animate = require("tailwindcss-animate")
const typography = require("@tailwindcss/typography")

/** @type {import('tailwindcss').Config} */
module.exports = {
  mode: 'jit',
  darkMode: "class",
  safelist: ["dark"],
  prefix: "",

  content: [
    './apps/main/src/**/*.{js,ts,vue}',
    './apps/widget/src/**/*.{js,ts,vue}',
    './shared-ui/**/*.{js,ts,vue}',
  ],

  theme: {
    container: {
      center: true,
      padding: '2rem',
      screens: {
        '2xl': '1400px'
      }
    },
    extend: {
      height: {
        screen: '100dvh'
      },
      minHeight: {
        screen: '100dvh'
      },
      colors: {
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        sidebar: {
          DEFAULT: 'hsl(var(--sidebar-background))',
          foreground: 'hsl(var(--sidebar-foreground))',
          primary: 'hsl(var(--sidebar-primary))',
          'primary-foreground': 'hsl(var(--sidebar-primary-foreground))',
          accent: 'hsl(var(--sidebar-accent))',
          'accent-foreground': 'hsl(var(--sidebar-accent-foreground))',
          border: 'hsl(var(--sidebar-border))',
          ring: 'hsl(var(--sidebar-ring))'
        },
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))'
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))'
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))'
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))'
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))'
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))'
        },
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))'
        }
      },
      borderRadius: {
        xl: 'calc(var(--radius) + 4px)',
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)'
      },
      keyframes: {
        'dot-flashing': {
          '0%': { opacity: '0.2' },
          '100%': { opacity: '1' },
        },
        'accordion-down': {
          from: {
            height: 0
          },
          to: {
            height: 'var(--radix-accordion-content-height)'
          }
        },
        shake: {
          '0%': { transform: 'translateX(0)' },
          '15%': { transform: 'translateX(-5px)' },
          '25%': { transform: 'translateX(5px)' },
          '35%': { transform: 'translateX(-5px)' },
          '45%': { transform: 'translateX(5px)' },
          '55%': { transform: 'translateX(-5px)' },
          '65%': { transform: 'translateX(5px)' },
          '75%': { transform: 'translateX(-5px)' },
          '85%': { transform: 'translateX(5px)' },
          '95%': { transform: 'translateX(-5px)' },
          '100%': { transform: 'translateX(0)' },
        },
        'accordion-up': {
          from: {
            height: 'var(--radix-accordion-content-height)'
          },
          to: {
            height: 0
          }
        },
        'collapsible-down': {
          from: {
            height: 0
          },
          to: {
            height: 'var(--radix-collapsible-content-height)'
          }
        },
        'collapsible-up': {
          from: {
            height: 'var(--radix-collapsible-content-height)'
          },
          to: {
            height: 0
          }
        },
        'fade-in-down': {
          '0%': {
            opacity: '0',
            transform: 'translateY(-3px)'
          },
          '100%': {
            opacity: '1',
            transform: 'translateY(0)'
          }
        },
        'bounce-in': {
          '0%': {
            transform: 'scale(0)'
          },
          '50%': {
            transform: 'scale(1.2)'
          },
          '100%': {
            transform: 'scale(1)'
          }
        },
        'fade-in': {
          '0%': {
            opacity: '0'
          },
          '100%': {
            opacity: '1'
          }
        },
        'fade-out': {
          '0%': {
            opacity: '1'
          },
          '100%': {
            opacity: '0'
          }
        },
        'slide-in': {
          '0%': {
            transform: 'translateY(10px)',
            opacity: '0'
          },
          '100%': {
            transform: 'translateY(0)',
            opacity: '1'
          }
        },
        'slide-out': {
          '0%': {
            transform: 'translateY(0)',
            opacity: '1'
          },
          '100%': {
            transform: 'translateY(10px)',
            opacity: '0'
          }
        }
      },
      animation: {
        'accordion-down': 'accordion-down 0.2s ease-out',
        'accordion-up': 'accordion-up 0.2s ease-out',
        'collapsible-down': 'collapsible-down 0.2s ease-in-out',
        'collapsible-up': 'collapsible-up 0.2s ease-in-out',
        'fade-in-down': 'fade-in-down 0.3s ease-out',
        'bounce-in': 'bounce-in 0.3s',
        'bounce-out': 'bounce-in 0.3s reverse',
        'fade-in': 'fade-in 0.3s ease-out',
        'fade-out': 'fade-out 0.3s ease-in',
        'slide-in': 'slide-in 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
        'slide-out': 'slide-out 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
        'shake': 'shake 0.5s infinite',
        'dot-flashing': 'dot-flashing 1s infinite linear alternate',
      }
    }
  },
  plugins: [animate, typography],
}
