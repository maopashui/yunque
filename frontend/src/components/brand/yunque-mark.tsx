import { cn } from '@/lib/utils'

/** Ascending skylark path (viewBox 0 0 24 24). */
const SKYLARK_PATH =
  'M7.39 2.40L8.94 4.57L11.82 7.81L13.27 9.80L13.74 10.27L14.01 10.34L14.25 10.22L14.71 9.79L15.68 8.67L16.03 8.40L16.49 8.19L17.33 8.08L18.28 8.31L19.39 8.30L18.72 8.81L18.19 9.42L17.70 10.43L16.71 13.02L16.28 13.82L15.63 14.68L14.26 15.98L13.38 16.60L11.60 17.66L10.75 18.39L9.33 19.98L7.44 22.43L7.13 22.75L6.75 23.00L6.67 22.92L6.66 22.71L6.95 21.43L7.31 20.30L7.24 20.07L6.59 20.09L4.93 20.64L4.70 20.67L4.61 20.62L4.65 20.49L5.16 19.97L8.77 17.05L9.92 15.39L10.19 14.64L10.07 14.07L9.06 13.10L8.03 11.95L7.29 10.97L6.71 10.01L6.44 9.43L6.42 9.11L5.66 7.71L5.71 7.55L6.21 7.58L6.33 7.45L5.40 6.19L5.11 5.58L5.11 5.41L5.37 5.42L6.48 6.11L6.88 6.27L6.75 5.96L5.64 4.33L5.37 3.70L5.27 3.23L5.30 3.16L5.43 3.18L6.66 4.32L7.15 4.68L7.45 4.80L7.44 4.58L7.08 3.83L6.85 3.07L6.49 1.00Z'

type YunQueMarkProps = {
  className?: string
  title?: string
}

/** Ascending skylark mark — monochrome, inherits currentColor. */
export function YunQueMark({ className, title }: YunQueMarkProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      className={cn('shrink-0', className)}
      aria-hidden={title ? undefined : true}
      role={title ? 'img' : undefined}
    >
      {title ? <title>{title}</title> : null}
      <path fill="currentColor" d={SKYLARK_PATH} />
    </svg>
  )
}

type YunQueBadgeProps = {
  className?: string
  markClassName?: string
  title?: string
}

/** Brand badge: dark tile + light skylark (sidebar / auth). */
export function YunQueBadge({ className, markClassName, title }: YunQueBadgeProps) {
  return (
    <div
      className={cn(
        'flex shrink-0 items-center justify-center rounded-md bg-zinc-900 text-zinc-50',
        className,
      )}
    >
      <YunQueMark className={cn('h-[70%] w-[70%]', markClassName)} title={title} />
    </div>
  )
}
