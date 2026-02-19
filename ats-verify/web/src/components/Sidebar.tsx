import { NavLink, useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import {
    LayoutDashboard,
    Package,
    Search,
    MapPin,
    Smartphone,
    ShieldAlert,
    Upload,
    LogOut,
    CheckCircle2,
    Ticket,
} from 'lucide-react';
import type { UserRole } from '../types';

interface NavItem {
    to: string;
    label: string;
    icon: React.ReactNode;
    roles: UserRole[];
}

const NAV_ITEMS: NavItem[] = [
    { to: '/', label: 'Дашборд', icon: <LayoutDashboard size={18} />, roles: ['admin', 'ats_staff', 'customs_staff', 'marketplace_staff', 'paid_user'] },
    { to: '/parcels', label: 'Посылки', icon: <Package size={18} />, roles: ['admin', 'customs_staff'] },
    { to: '/upload', label: 'Загрузка CSV', icon: <Upload size={18} />, roles: ['marketplace_staff'] },
    { to: '/track', label: 'Поиск трека', icon: <Search size={18} />, roles: ['ats_staff', 'admin'] },
    { to: '/tracking', label: 'Отслеживание', icon: <MapPin size={18} />, roles: ['admin', 'ats_staff', 'customs_staff', 'marketplace_staff', 'paid_user'] },
    { to: '/imei', label: 'IMEI Проверка', icon: <Smartphone size={18} />, roles: ['customs_staff', 'paid_user'] },
    { to: '/risks', label: 'Риски', icon: <ShieldAlert size={18} />, roles: ['admin', 'customs_staff'] },
    { to: '/tickets', label: 'Тикеты', icon: <Ticket size={18} />, roles: ['ats_staff', 'customs_staff', 'admin'] },
];

const ROLE_LABELS: Record<UserRole, string> = {
    admin: 'Администратор',
    ats_staff: 'Сотрудник АТС',
    customs_staff: 'Сотрудник таможни',
    marketplace_staff: 'Маркетплейс',
    paid_user: 'Пользователь',
};

export default function Sidebar() {
    const { user, logout } = useAuth();
    const navigate = useNavigate();

    const handleLogout = () => {
        logout();
        navigate('/login');
    };

    const filteredItems = NAV_ITEMS.filter((item) =>
        user ? item.roles.includes(user.role) : false
    );

    return (
        <aside className="sidebar">
            {/* Logo */}
            <div className="sidebar-logo">
                <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
                    <CheckCircle2 size={18} className="text-white" />
                </div>
                <div>
                    <span className="text-base font-bold text-text-primary">ATS-Verify</span>
                </div>
            </div>

            {/* Navigation */}
            <nav className="flex-1 py-3 space-y-0.5 overflow-y-auto">
                {filteredItems.map((item) => (
                    <NavLink
                        key={item.to}
                        to={item.to}
                        end={item.to === '/'}
                        className={({ isActive }) =>
                            `nav-link ${isActive ? 'nav-link-active' : ''}`
                        }
                    >
                        {item.icon}
                        {item.label}
                    </NavLink>
                ))}
            </nav>

            {/* User Info */}
            {user && (
                <div className="border-t border-border p-4">
                    <div className="flex items-center gap-3 mb-3">
                        <div className="w-9 h-9 rounded-full bg-primary-light flex items-center justify-center text-primary font-semibold text-sm">
                            {user.username.slice(0, 2).toUpperCase()}
                        </div>
                        <div className="flex-1 min-w-0">
                            <p className="text-sm font-medium text-text-primary truncate">{user.username}</p>
                            <p className="text-xs text-text-muted">{ROLE_LABELS[user.role]}</p>
                        </div>
                    </div>
                    <button
                        onClick={handleLogout}
                        className="flex items-center gap-2 text-sm text-text-muted hover:text-danger transition-colors w-full cursor-pointer"
                    >
                        <LogOut size={15} />
                        Выйти
                    </button>
                </div>
            )}
        </aside>
    );
}
