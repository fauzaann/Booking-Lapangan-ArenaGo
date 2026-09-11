import { BrowserRouter, Routes, Route } from "react-router-dom";
import { AuthProvider } from "./context/AuthContext";
import RequireAuth from "./components/routing/RequireAuth";
import AppLayout from "./layouts/AppLayout";
import AuthLayout from "./layouts/AuthLayout";
import Explore from "./pages/Explore";
import SelectSlot from "./pages/SelectSlot";
import Confirmation from "./pages/Confirmation";
import Payment from "./pages/Payment";
import PaymentResult from "./pages/PaymentResult";
import ETicket from "./pages/ETicket";
import MyBookings from "./pages/MyBookings";
import Profile from "./pages/Profile";
import Login from "./pages/Login";
import Register from "./pages/Register";
import TransactionList from "./pages/TransactionList";
import TransactionDetail from "./pages/TransactionDetail";
import AdminDashboard from "./pages/AdminDashboard";

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route element={<AuthLayout />}>
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
          </Route>

          <Route element={<AppLayout />}>
            {/* Public — browsing doesn't require an account */}
            <Route path="/" element={<Explore />} />
            <Route path="/jadwal/:courtId" element={<SelectSlot />} />

            {/* Requires login — booking, payment, personal data */}
            <Route
              path="/konfirmasi"
              element={
                <RequireAuth>
                  <Confirmation />
                </RequireAuth>
              }
            />
            <Route
              path="/pembayaran"
              element={
                <RequireAuth>
                  <Payment />
                </RequireAuth>
              }
            />
            <Route
              path="/payment/success"
              element={<RequireAuth><PaymentResult /></RequireAuth>}
            />
            <Route
              path="/payment/failed"
              element={<RequireAuth><PaymentResult /></RequireAuth>}
            />
            <Route
              path="/e-tiket"
              element={
                <RequireAuth>
                  <ETicket />
                </RequireAuth>
              }
            />
            <Route
              path="/riwayat"
              element={
                <RequireAuth>
                  <MyBookings />
                </RequireAuth>
              }
            />
            <Route
              path="/profil"
              element={
                <RequireAuth>
                  <Profile />
                </RequireAuth>
              }
            />

            {/* Admin only */}
            <Route
              path="/admin"
              element={
                <RequireAuth role="admin">
                  <AdminDashboard />
                </RequireAuth>
              }
            />
            <Route
              path="/admin/transaksi"
              element={
                <RequireAuth role="admin">
                  <TransactionList />
                </RequireAuth>
              }
            />
            <Route
              path="/admin/transaksi/:id"
              element={
                <RequireAuth role="admin">
                  <TransactionDetail />
                </RequireAuth>
              }
            />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}
