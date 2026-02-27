import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './hooks/useAuth';
import Layout from './components/Layout';
import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';
import DashboardPage from './pages/DashboardPage';
import ParcelsPage from './pages/ParcelsPage';
import UploadPage from './pages/UploadPage';
import TrackPage from './pages/TrackPage';
import TrackingPage from './pages/TrackingPage';
import IMEIPage from './pages/IMEIPage';
import RisksPage from './pages/RisksPage';
import TicketsPage from './pages/TicketsPage';
import AnalyticsPage from './pages/AnalyticsPage';
import type { UserRole } from './types';
import { Toaster } from 'react-hot-toast';
import './index.css';

// Role-based route guard
function RoleGuard({ children, allowed }: { children: React.ReactNode; allowed: UserRole[] }) {
  const { user } = useAuth();
  if (!user || !allowed.includes(user.role)) {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}

function App() {
  return (
    <BrowserRouter>
      <Toaster position="top-right" />
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route element={<Layout />}>
            <Route index element={<DashboardPage />} />

            <Route path="parcels" element={
              <RoleGuard allowed={['admin', 'customs_staff']}>
                <ParcelsPage />
              </RoleGuard>
            } />

            <Route path="upload" element={
              <RoleGuard allowed={['marketplace_staff', 'admin']}>
                <UploadPage />
              </RoleGuard>
            } />

            <Route path="track" element={
              <RoleGuard allowed={['ats_staff', 'admin']}>
                <TrackPage />
              </RoleGuard>
            } />

            <Route path="tracking" element={<TrackingPage />} />

            <Route path="imei" element={
              <RoleGuard allowed={['customs_staff', 'paid_user']}>
                <IMEIPage />
              </RoleGuard>
            } />

            <Route path="risks" element={
              <RoleGuard allowed={['admin', 'customs_staff']}>
                <RisksPage />
              </RoleGuard>
            } />

            <Route path="analytics" element={
              <RoleGuard allowed={['admin', 'ats_staff', 'customs_staff']}>
                <AnalyticsPage />
              </RoleGuard>
            } />

            <Route path="tickets" element={
              <RoleGuard allowed={['ats_staff', 'customs_staff', 'admin']}>
                <TicketsPage />
              </RoleGuard>
            } />
          </Route>

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;
