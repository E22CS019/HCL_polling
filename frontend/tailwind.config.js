/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          50:  '#f0f4ff',
          100: '#dde5ff',
          200: '#c3d0ff',
          300: '#9db0fd',
          400: '#7488f9',
          500: '#5563f3',
          600: '#4145e8',
          700: '#3535cc',
          800: '#2c2fa5',
          900: '#2a2d83',
          950: '#1a1b4e',
        },
      },
      animation: {
        'bar-grow': 'barGrow 0.5s ease-out forwards',
        'fade-in':  'fadeIn 0.3s ease-out',
        'pulse-dot': 'pulseDot 1.4s ease-in-out infinite',
      },
      keyframes: {
        barGrow: {
          from: { width: '0%' },
          to:   { width: 'var(--bar-width)' },
        },
        fadeIn: {
          from: { opacity: '0', transform: 'translateY(8px)' },
          to:   { opacity: '1', transform: 'translateY(0)' },
        },
        pulseDot: {
          '0%, 80%, 100%': { transform: 'scale(0)' },
          '40%':           { transform: 'scale(1)' },
        },
      },
    },
  },
  plugins: [],
}
