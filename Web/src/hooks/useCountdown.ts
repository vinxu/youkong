import { useState, useCallback, useRef, useEffect } from 'react'

export function useCountdown(initialSeconds: number = 60) {
  const [seconds, setSeconds] = useState(0)
  const timerRef = useRef<number>()

  const start = useCallback(() => {
    // 清除之前的定时器
    if (timerRef.current) {
      clearInterval(timerRef.current)
    }

    setSeconds(initialSeconds)
    timerRef.current = window.setInterval(() => {
      setSeconds((prev) => {
        if (prev <= 1) {
          clearInterval(timerRef.current)
          return 0
        }
        return prev - 1
      })
    }, 1000)
  }, [initialSeconds])

  const reset = useCallback(() => {
    if (timerRef.current) {
      clearInterval(timerRef.current)
    }
    setSeconds(0)
  }, [])

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current)
      }
    }
  }, [])

  return {
    seconds,
    start,
    reset,
    isActive: seconds > 0,
  }
}

export default useCountdown
