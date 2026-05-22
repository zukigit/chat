import { createRoot } from 'react-dom/client'
import { BrowserRouter, Route, Routes } from 'react-router-dom'
import './global.css'
import './theme.css'
import './ui.css'
import PrivateRoute from './components/PrivateRoute'
import SetupPage from './pages/SetupPage'
import LoginPage from './pages/LoginPage'
import SignupPage from './pages/SignupPage'
import CallbackPage from './pages/CallbackPage'
import KeySetupPage from './pages/KeySetupPage'
import PinEntryPage from './pages/PinEntryPage'
import HomePage from './pages/HomePage'

createRoot(document.getElementById('root')!).render(
  <BrowserRouter>
    <Routes>
      {/* Public routes */}
      <Route path="/login" element={<LoginPage />} />
      <Route path="/signup" element={<SignupPage />} />
      <Route path="/setup" element={<SetupPage />} />
      <Route path="/callback" element={<CallbackPage />} />

      {/* Private routes */}
      <Route element={<PrivateRoute />}>
        <Route path="/key-setup" element={<KeySetupPage />} />
        <Route path="/pin-entry" element={<PinEntryPage />} />
        <Route path="/" element={<HomePage />} />
      </Route>
    </Routes>
  </BrowserRouter>
)
