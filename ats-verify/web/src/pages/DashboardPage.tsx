import { useState, useEffect } from 'react';
import { useAuth } from '../hooks/useAuth';
import { Link } from 'react-router-dom';
import {
    Package,
    Upload,
    ShieldAlert,
    Smartphone,
    Search,
    ArrowUpRight,
    Ticket,
    AlertCircle,
    ArrowRightCircle,
    CheckCircle2,
    TrendingUp,
    Activity,
} from 'lucide-react';
import api from '../lib/api';
import type { UserRole, SupportTicket } from '../types';

interface StatCard {
    label: string;
    value: string;
    change?: string;
    changeType?: 'up' | 'down' | 'neutral';
    icon: React.ReactNode;
    iconBg: string;
    roles: UserRole[];
}

function getGreeting(): string {
    const h = new Date().getHours();
    if (h < 6) return 'Доброй ночи';
    if (h < 12) return 'Доброе утро';
    if (h < 18) return 'Добрый день';
    return 'Добрый вечер';
}

const STATS: StatCard[] = [
    { label: 'Всего посылок', value: '—', change: '+5.2%', changeType: 'up', icon: <Package size={20} />, iconBg: 'bg-blue-50 text-blue-600', roles: ['admin', 'customs_staff'] },
    { label: 'Загружено сегодня', value: '—', icon: <Upload size={20} />, iconBg: 'bg-violet-50 text-violet-600', roles: ['admin', 'marketplace_staff'] },
    { label: 'Красный риск', value: '—', icon: <ShieldAlert size={20} />, iconBg: 'bg-red-50 text-red-600', roles: ['admin', 'customs_staff'] },
    { label: 'IMEI проверок', value: '—', icon: <Smartphone size={20} />, iconBg: 'bg-emerald-50 text-emerald-600', roles: ['admin', 'customs_staff', 'paid_user'] },
    { label: 'Поисков трека', value: '—', icon: <Search size={20} />, iconBg: 'bg-amber-50 text-amber-600', roles: ['admin', 'ats_staff'] },
    { label: 'Активных тикетов', value: '—', icon: <Ticket size={20} />, iconBg: 'bg-cyan-50 text-cyan-600', roles: ['admin', 'ats_staff', 'customs_staff'] },
];

interface QuickAction {
    label: string;
    description: string;
    to: string;
    icon: React.ReactNode;
    roles: UserRole[];
}

const ACTIONS: QuickAction[] = [
    { label: 'Загрузить CSV', description: 'Импорт файлов маркетплейса', to: '/upload', icon: <Upload size={18} />, roles: ['marketplace_staff'] },
    { label: 'Поиск трек-номера', description: 'Отслеживание посылок', to: '/track', icon: <Search size={18} />, roles: ['ats_staff', 'admin'] },
    { label: 'Проверка IMEI', description: 'Верификация IMEI по PDF', to: '/imei', icon: <Smartphone size={18} />, roles: ['customs_staff', 'paid_user'] },
    { label: 'Управление рисками', description: 'Анализ ИИН/БИН', to: '/risks', icon: <ShieldAlert size={18} />, roles: ['admin', 'customs_staff'] },
    { label: 'Канбан тикеты', description: 'Обработка обращений', to: '/tickets', icon: <Ticket size={18} />, roles: ['ats_staff', 'customs_staff', 'admin'] },
];

// Ticket status breakdown widget
function TicketBreakdown({ tickets }: { tickets: SupportTicket[] }) {
    const toDo = tickets.filter((t) => t.status === 'to_do').length;
    const inProgress = tickets.filter((t) => t.status === 'in_progress').length;
    const completed = tickets.filter((t) => t.status === 'completed').length;
    const total = tickets.length || 1;

    const segments = [
        { count: toDo, label: 'К выполнению', color: 'bg-blue-500', icon: <AlertCircle size={14} className="text-blue-500" /> },
        { count: inProgress, label: 'В работе', color: 'bg-amber-500', icon: <ArrowRightCircle size={14} className="text-amber-500" /> },
        { count: completed, label: 'Завершено', color: 'bg-green-500', icon: <CheckCircle2 size={14} className="text-green-500" /> },
    ];

    return (
        <div className="card p-5">
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-text-primary">Статус тикетов</h3>
                <Link to="/tickets" className="text-xs text-primary hover:text-primary-dark flex items-center gap-1">
                    Перейти <ArrowUpRight size={12} />
                </Link>
            </div>
            {/* Progress bar */}
            <div className="flex rounded-full h-2.5 overflow-hidden mb-4 bg-bg-muted">
                {segments.map((seg) => (
                    <div
                        key={seg.label}
                        className={`${seg.color} transition-all duration-500`}
                        style={{ width: `${(seg.count / total) * 100}%` }}
                    />
                ))}
            </div>
            <div className="grid grid-cols-3 gap-3">
                {segments.map((seg) => (
                    <div key={seg.label} className="text-center">
                        <div className="flex items-center justify-center gap-1 mb-1">
                            {seg.icon}
                            <span className="text-xl font-bold text-text-primary">{seg.count}</span>
                        </div>
                        <p className="text-[11px] text-text-muted">{seg.label}</p>
                    </div>
                ))}
            </div>
        </div>
    );
}

// Activity / recent items widget
function RecentActivity({ tickets }: { tickets: SupportTicket[] }) {
    const recent = [...tickets]
        .sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
        .slice(0, 5);

    return (
        <div className="card p-5">
            <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-text-primary flex items-center gap-2">
                    <Activity size={16} className="text-primary" /> Последняя активность
                </h3>
            </div>
            {recent.length === 0 ? (
                <p className="text-sm text-text-muted text-center py-6">Нет активности</p>
            ) : (
                <div className="space-y-3">
                    {recent.map((t) => (
                        <Link key={t.id} to="/tickets" className="flex items-start gap-3 group">
                            <div className={`mt-0.5 w-2 h-2 rounded-full shrink-0 ${t.priority === 'high' ? 'bg-danger' : t.priority === 'medium' ? 'bg-warning' : 'bg-blue-400'}`} />
                            <div className="flex-1 min-w-0">
                                <p className="text-sm text-text-primary group-hover:text-primary transition-colors truncate">{t.rejection_reason}</p>
                                <p className="text-[11px] text-text-muted">{t.iin} · {new Date(t.updated_at).toLocaleDateString('ru-RU')}</p>
                            </div>
                            <span className={`shrink-0 text-[10px] px-1.5 py-0.5 rounded-full ${t.status === 'completed' ? 'bg-success-light text-green-700' : t.status === 'in_progress' ? 'bg-warning-light text-amber-700' : 'bg-info-light text-blue-700'}`}>
                                {t.status === 'to_do' ? 'Новый' : t.status === 'in_progress' ? 'В работе' : 'Готово'}
                            </span>
                        </Link>
                    ))}
                </div>
            )}
        </div>
    );
}

export default function DashboardPage() {
    const { user } = useAuth();
    const [tickets, setTickets] = useState<SupportTicket[]>([]);

    useEffect(() => {
        if (!user) return;
        if (['ats_staff', 'customs_staff', 'admin'].includes(user.role)) {
            api.get('/tickets').then((res) => setTickets(res.data || [])).catch(() => { });
        }
    }, [user]);

    if (!user) return null;

    const userStats = STATS.filter((s) => s.roles.includes(user.role));
    const userActions = ACTIONS.filter((a) => a.roles.includes(user.role));
    const showTicketWidgets = ['ats_staff', 'customs_staff', 'admin'].includes(user.role);

    // Fill ticket count
    const statsWithData = userStats.map((s) => {
        if (s.label === 'Активных тикетов') {
            return { ...s, value: String(tickets.filter((t) => t.status !== 'completed').length) };
        }
        if (s.label === 'Красный риск') {
            return { ...s, value: String(tickets.filter((t) => t.risk_level === 'red').length), change: undefined };
        }
        return s;
    });

    return (
        <div>
            {/* Welcome Header */}
            <div className="mb-8">
                <div className="flex items-center gap-2 mb-1">
                    <TrendingUp size={20} className="text-primary" />
                    <span className="text-sm text-text-muted">Обзор системы</span>
                </div>
                <h1 className="text-3xl font-bold text-text-primary">{getGreeting()}, {user.username} 👋</h1>
            </div>

            {/* Stat Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-8">
                {statsWithData.map((stat) => (
                    <div key={stat.label} className="stat-card group hover:border-primary/30 transition-colors">
                        <div className="flex items-center justify-between">
                            <div className={`w-10 h-10 rounded-xl ${stat.iconBg} flex items-center justify-center`}>
                                {stat.icon}
                            </div>
                            {stat.change && (
                                <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${stat.changeType === 'up' ? 'bg-success-light text-green-700' : 'bg-danger-light text-red-700'
                                    }`}>
                                    {stat.change}
                                </span>
                            )}
                        </div>
                        <div>
                            <p className="stat-value">{stat.value}</p>
                            <p className="stat-label">{stat.label}</p>
                        </div>
                    </div>
                ))}
            </div>

            {/* Widgets Row */}
            {showTicketWidgets && (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-8">
                    <TicketBreakdown tickets={tickets} />
                    <RecentActivity tickets={tickets} />
                </div>
            )}

            {/* Quick Actions */}
            {userActions.length > 0 && (
                <div className="card p-6">
                    <h2 className="text-sm font-semibold text-text-primary mb-4 flex items-center gap-2">
                        <ArrowUpRight size={16} className="text-primary" /> Быстрые действия
                    </h2>
                    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                        {userActions.map((action) => (
                            <Link
                                key={action.to}
                                to={action.to}
                                className="flex items-center gap-3.5 px-4 py-3.5 rounded-xl border border-border hover:border-primary/30 hover:bg-primary-50 transition-all group"
                            >
                                <div className="w-9 h-9 rounded-lg bg-primary-50 flex items-center justify-center text-primary group-hover:bg-primary group-hover:text-white transition-all">
                                    {action.icon}
                                </div>
                                <div className="flex-1 min-w-0">
                                    <p className="text-sm font-medium text-text-primary group-hover:text-primary transition-colors">{action.label}</p>
                                    <p className="text-[11px] text-text-muted truncate">{action.description}</p>
                                </div>
                                <ArrowUpRight size={14} className="text-text-muted group-hover:text-primary transition-colors shrink-0" />
                            </Link>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
}
