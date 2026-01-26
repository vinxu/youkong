import type { Availability } from '../types'
import Avatar from './Avatar'

interface AvailabilityCardProps {
  availability: Availability
  showUser?: boolean
}

function formatTime(startTime: string, endTime: string): string {
  const start = new Date(startTime)
  const end = new Date(endTime)
  const now = new Date()

  const isToday = start.toDateString() === now.toDateString()
  const isTomorrow = start.toDateString() === new Date(now.getTime() + 86400000).toDateString()

  const formatHour = (date: Date) => {
    const hours = date.getHours()
    const minutes = date.getMinutes()
    return minutes === 0 ? `${hours}:00` : `${hours}:${minutes.toString().padStart(2, '0')}`
  }

  let datePrefix = ''
  if (isToday) {
    datePrefix = '今天'
  } else if (isTomorrow) {
    datePrefix = '明天'
  } else {
    datePrefix = `${start.getMonth() + 1}/${start.getDate()}`
  }

  return `${datePrefix} ${formatHour(start)} - ${formatHour(end)}`
}

function getLocationEmoji(type: string): string {
  switch (type) {
    case 'PRESET':
      return ''
    case 'FLEXIBLE':
      return ''
    case 'CUSTOM':
      return ''
    default:
      return ''
  }
}

export function AvailabilityCard({ availability, showUser = true }: AvailabilityCardProps) {
  const { user, startTime, endTime, location } = availability
  const locationEmoji = getLocationEmoji(location.type)
  const locationName = location.name || (location.type === 'FLEXIBLE' ? '灵活安排' : '待定')

  return (
    <div className="bg-white rounded-2xl p-4 shadow-sm border border-gray-100">
      {showUser && (
        <div className="flex items-center gap-3 mb-3">
          <Avatar src={user.avatar} name={user.nickname} size="sm" />
          <span className="font-medium text-gray-900">{user.nickname}</span>
        </div>
      )}

      <div className="space-y-2">
        <div className="flex items-center gap-2 text-gray-700">
          <span className="text-lg">📅</span>
          <span>{formatTime(startTime, endTime)}</span>
        </div>

        <div className="flex items-center gap-2 text-gray-700">
          <span className="text-lg">📍</span>
          <span>
            {locationEmoji && `${locationEmoji} `}
            {locationName}
          </span>
        </div>
      </div>
    </div>
  )
}

export default AvailabilityCard
