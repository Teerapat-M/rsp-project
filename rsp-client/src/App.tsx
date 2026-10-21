import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { PageStatus } from './components/PageStatus'
import { AuthProvider, useAuth } from './context/AuthContext'
import { GameProvider } from './context/GameContext'
import { SplashPage } from './pages/SplashPage'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { ForgotPage } from './pages/ForgotPage'
import { ResetPage } from './pages/ResetPage'
import { GuestPage } from './pages/GuestPage'
import { LobbyPage } from './pages/LobbyPage'
import { CreateRoomPage } from './pages/CreateRoomPage'
import { WaitingRoomPage } from './pages/WaitingRoomPage'
import { GamePage } from './pages/GamePage'
import { MatchResultPage } from './pages/MatchResultPage'
import { StatsPage } from './pages/StatsPage'
import { HistoryPage } from './pages/HistoryPage'
import { MatchDetailPage } from './pages/MatchDetailPage'
import type { ReactNode } from 'react'

function Gate({ children, registered }: { children: ReactNode; registered?: boolean }) {
  const { user, ready } = useAuth()
  if (!ready) return <PageStatus>Loading…</PageStatus>
  if (!user) return <Navigate to="/login" replace />
  if (registered && user.isGuest) return <Navigate to="/register" replace />
  return children
}

export default function App() {
  return (
    <AuthProvider>
      <GameProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<SplashPage />} />
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/forgot" element={<ForgotPage />} />
            <Route path="/reset" element={<ResetPage />} />
            <Route path="/guest" element={<GuestPage />} />
            <Route
              path="/lobby"
              element={
                <Gate>
                  <LobbyPage />
                </Gate>
              }
            />
            <Route
              path="/create"
              element={
                <Gate>
                  <CreateRoomPage />
                </Gate>
              }
            />
            <Route
              path="/room/:id"
              element={
                <Gate>
                  <WaitingRoomPage />
                </Gate>
              }
            />
            <Route
              path="/play"
              element={
                <Gate>
                  <GamePage />
                </Gate>
              }
            />
            <Route
              path="/result"
              element={
                <Gate>
                  <MatchResultPage />
                </Gate>
              }
            />
            <Route
              path="/stats"
              element={
                <Gate registered>
                  <StatsPage />
                </Gate>
              }
            />
            <Route
              path="/history"
              element={
                <Gate registered>
                  <HistoryPage />
                </Gate>
              }
            />
            <Route
              path="/history/:id"
              element={
                <Gate registered>
                  <MatchDetailPage />
                </Gate>
              }
            />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </GameProvider>
    </AuthProvider>
  )
}
