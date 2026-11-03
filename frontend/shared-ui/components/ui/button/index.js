import { cva } from 'class-variance-authority'

export { default as Button } from './Button.vue'

export const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-[color,background-color,border-color,box-shadow,transform] duration-150 ease-out motion-safe:[&:active:not(:focus-visible)]:scale-[0.97] motion-reduce:transition-none focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0',
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground shadow-sm can-hover:hover:bg-primary/90',
        destructive:
          'bg-destructive text-destructive-foreground shadow-sm can-hover:hover:bg-destructive/90',
        outline:
          'border border-input bg-background shadow-sm can-hover:hover:bg-accent can-hover:hover:text-accent-foreground',
        secondary:
          'bg-secondary text-secondary-foreground shadow-sm can-hover:hover:bg-secondary/80',
        ghost: 'can-hover:hover:bg-accent can-hover:hover:text-accent-foreground',
        link: 'text-muted-foreground underline underline-offset-4 can-hover:hover:text-foreground active:scale-100'
      },
      size: {
        default: 'h-9 px-4 py-2',
        xs: 'h-7 rounded-md px-2',
        sm: 'h-8 rounded-md px-3 text-xs',
        lg: 'h-10 rounded-md px-8',
        icon: 'h-9 w-9'
      }
    },
    defaultVariants: {
      variant: 'default',
      size: 'default'
    }
  }
)
