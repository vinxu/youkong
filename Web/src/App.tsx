import { useEffect } from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import {
  LandingPage,
  InvitePage,
  WelcomePage,
  AvailabilityPage,
  ExpiredPage,
  NotFoundPage,
} from './pages'
import { useAuthStore } from './stores/authStore'

function App() {
  const loadFromStorage = useAuthStore((state) => state.loadFromStorage)

  useEffect(() => {
    loadFromStorage()
  }, [loadFromStorage])

  return (
    <BrowserRouter>
      <Routes>
        {/* 邀请落地页 */}
        <Route path="/i/:code" element={<InvitePage />} />

        {/* 欢迎页 */}
        <Route path="/welcome" element={<WelcomePage />} />

        {/* 有空状态页 */}
        <Route path="/availability/:userId" element={<AvailabilityPage />} />
        <Route path="/availability" element={<AvailabilityPage />} />

        {/* 邀请失效页 */}
        <Route path="/expired" element={<ExpiredPage />} />

        {/* 首页 - Landing Page */}
        <Route path="/" element={<LandingPage />} />

        {/* 404 */}
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </BrowserRouter>
  )
}

export default App
