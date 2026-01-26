import { InputHTMLAttributes, forwardRef } from 'react'

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  error?: string
  suffix?: React.ReactNode
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ error, suffix, className = '', ...props }, ref) => {
    return (
      <div className="w-full">
        <div className="relative">
          <input
            ref={ref}
            className={`
              w-full px-4 py-3 rounded-xl border
              text-gray-900 placeholder-gray-400
              transition-all duration-200
              focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent
              ${error ? 'border-red-500' : 'border-gray-200'}
              ${suffix ? 'pr-24' : ''}
              ${className}
            `}
            {...props}
          />
          {suffix && (
            <div className="absolute right-2 top-1/2 -translate-y-1/2">
              {suffix}
            </div>
          )}
        </div>
        {error && (
          <p className="mt-1 text-sm text-red-500">{error}</p>
        )}
      </div>
    )
  }
)

Input.displayName = 'Input'

export default Input
