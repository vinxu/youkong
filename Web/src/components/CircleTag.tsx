interface CircleTagProps {
  name: string
  emoji: string
  color?: string
  size?: 'sm' | 'md'
}

export function CircleTag({ name, emoji, color = '#10b981', size = 'md' }: CircleTagProps) {
  const sizeClasses = {
    sm: 'px-2 py-1 text-xs',
    md: 'px-3 py-1.5 text-sm',
  }

  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full font-medium ${sizeClasses[size]}`}
      style={{
        backgroundColor: `${color}15`,
        color: color,
      }}
    >
      <span>{emoji}</span>
      <span>{name}</span>
    </span>
  )
}

export default CircleTag
