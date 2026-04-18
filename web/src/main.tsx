import { createRoot } from 'react-dom/client'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import './global.css'
import PrivateRoute from './components/PrivateRoute'
import SetupPage from './pages/SetupPage'
import LoginPage from './pages/LoginPage'
import SignupPage from './pages/SignupPage'

createRoot(document.getElementById('root')!).render(
  <BrowserRouter>
    <Routes>
      {/* Public routes */}
      <Route path="/login" element={<LoginPage />} />
      <Route path="/signup" element={<SignupPage />} />
      <Route path="/setup" element={<SetupPage />} />

      {/* Private routes */}
      <Route element={<PrivateRoute />}>
        <Route path="/" element={<div>Chat (coming soon)</div>} />
      </Route>
    </Routes>
  </BrowserRouter>
)
