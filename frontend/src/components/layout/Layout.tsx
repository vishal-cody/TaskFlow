import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { LayoutDashboard, List, LogOut } from 'lucide-react';
import clsx from 'clsx';
import styles from './Layout.module.css';

export function Layout() {
  const navigate = useNavigate();

  const handleLogout = () => {
    localStorage.removeItem('token');
    navigate('/login');
  };

  return (
    <div className={styles.layout}>
      <aside className={styles.sidebar}>
        <div className={styles.logo}>
          TaskFlow
        </div>
        <nav className={styles.nav}>
          <NavLink 
            to="/" 
            className={({ isActive }) => clsx(styles.navItem, isActive && styles.active)}
          >
            <LayoutDashboard size={20} />
            Dashboard
          </NavLink>
          <NavLink 
            to="/jobs" 
            className={({ isActive }) => clsx(styles.navItem, isActive && styles.active)}
          >
            <List size={20} />
            All Jobs
          </NavLink>
        </nav>
        <button onClick={handleLogout} className={styles.logoutBtn}>
          <LogOut size={20} />
          Logout
        </button>
      </aside>
      
      <main className={styles.main}>
        <header className={styles.header}>
          <h2>Platform Dashboard</h2>
        </header>
        <div className={styles.content}>
          <Outlet />
        </div>
      </main>
    </div>
  );
}
