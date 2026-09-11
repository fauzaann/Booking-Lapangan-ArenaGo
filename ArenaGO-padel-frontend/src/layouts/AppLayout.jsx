import { Outlet } from "react-router-dom";
import Navbar from "../components/ui/Navbar";
import BottomNav from "../components/ui/BottomNav";
import ConciergeWidget from "../components/booking/ConciergeWidget";

export default function AppLayout() {
  return (
    <div className="min-h-screen bg-canvas">
      <Navbar />
      <main className="mx-auto max-w-[1440px] pb-24 md:pb-16">
        <Outlet />
      </main>
      <BottomNav />
      <ConciergeWidget />
    </div>
  );
}
