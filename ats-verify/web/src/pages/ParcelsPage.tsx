import { useState, useEffect, useCallback } from 'react';
import { Search, Download, Filter } from 'lucide-react';
import api from '../lib/api';
import { useAuth } from '../hooks/useAuth';
import type { Parcel } from '../types';

export default function ParcelsPage() {
    const [parcels, setParcels] = useState<Parcel[]>([]);
    const [searchQuery, setSearchQuery] = useState('');
    const [statusFilter, setStatusFilter] = useState<'all' | 'used' | 'unused'>('all');
    const [page, setPage] = useState(1);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(false);
    const { user } = useAuth();
    const limit = 20;

    const fetchParcels = useCallback(async () => {
        setLoading(true);
        try {
            const params: Record<string, string | number> = { page, limit };
            if (statusFilter !== 'all') params.status = statusFilter;
            if (searchQuery) params.search = searchQuery;

            const { data } = await api.get('/parcels', { params });
            setParcels(data.parcels || []);
            setTotal(data.total || 0);
        } catch {
            // If API unavailable, show empty state
            setParcels([]);
        } finally {
            setLoading(false);
        }
    }, [page, statusFilter, searchQuery]);

    useEffect(() => { fetchParcels(); }, [fetchParcels]);

    // Debounced search
    useEffect(() => {
        setPage(1);
    }, [searchQuery, statusFilter]);

    const totalPages = Math.ceil(total / limit) || 1;

    const handleMarkUsed = async (trackNumber: string) => {
        try {
            await api.post('/parcels/mark-used', { track_number: trackNumber });
            // Optimistically update the local state to show 'IsUsed: true'
            setParcels(prev => prev.map(p => p.track_number === trackNumber ? { ...p, is_used: true } : p));
        } catch (err: any) {
            console.error('Failed to mark parcel as used:', err);
            const msg = err.response?.data?.error || 'Ошибка: Не удалось отметить посылку как использованную.';
            alert(msg);
        }
    };

    const handleExportCSV = () => {
        if (parcels.length === 0) return;
        const headers = ['Трек-номер', 'Маркетплейс', 'Страна', 'Бренд', 'Товар', 'Статус', 'Дата загрузки'];
        const rows = parcels.map(p => [p.track_number, p.marketplace, p.country, p.brand, p.product_name, p.is_used ? 'Использован' : 'Не использован', p.upload_date.split('T')[0]]);
        const csv = [headers.join(','), ...rows.map(r => r.join(','))].join('\n');
        const blob = new Blob([csv], { type: 'text/csv' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a'); a.href = url; a.download = 'parcels.csv'; a.click();
    };

    return (
        <div>
            {/* Header */}
            <div className="flex items-center justify-between mb-6">
                <div>
                    <h1 className="page-title">Посылки</h1>
                    <p className="page-subtitle">Управление базой данных отправлений</p>
                </div>
                <button onClick={handleExportCSV} className="btn-secondary">
                    <Download size={16} />
                    Экспорт CSV
                </button>
            </div>

            {/* Filters */}
            <div className="card p-4 mb-4">
                <div className="flex items-center gap-3">
                    <div className="relative flex-1 max-w-sm">
                        <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
                        <input
                            type="text"
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className="input !pl-10"
                            placeholder="Поиск по треку или товару..."
                        />
                    </div>
                    <div className="flex items-center gap-1.5">
                        <Filter size={16} className="text-text-muted" />
                        {(['all', 'unused', 'used'] as const).map((f) => (
                            <button
                                key={f}
                                onClick={() => setStatusFilter(f)}
                                className={`px-3 py-1.5 text-xs font-medium rounded-lg border transition-all cursor-pointer ${statusFilter === f
                                    ? 'bg-primary text-white border-primary'
                                    : 'bg-bg-white border-border text-text-secondary hover:border-primary/30'
                                    }`}
                            >
                                {f === 'all' ? 'Все' : f === 'used' ? 'Использованные' : 'Неиспользованные'}
                            </button>
                        ))}
                    </div>
                </div>
            </div>

            {/* Table */}
            <div className="card overflow-hidden">
                {loading ? (
                    <div className="p-8 text-center text-text-muted">Загрузка...</div>
                ) : parcels.length === 0 ? (
                    <div className="p-8 text-center text-text-muted">Нет данных. Убедитесь что бэкенд запущен и загружены посылки.</div>
                ) : (
                    <table className="data-table">
                        <thead>
                            <tr>
                                <th>Трек-номер</th>
                                <th>Маркетплейс</th>
                                <th>Страна</th>
                                <th>Бренд</th>
                                <th>Товар</th>
                                <th>Статус</th>
                                <th>Дата загрузки</th>
                                <th>Действия</th>
                            </tr>
                        </thead>
                        <tbody>
                            {parcels.map((p) => (
                                <tr key={p.id}>
                                    <td className="font-mono text-text-primary font-medium">{p.track_number}</td>
                                    <td>{p.marketplace}</td>
                                    <td>{p.country}</td>
                                    <td>{p.brand}</td>
                                    <td>{p.product_name}</td>
                                    <td>
                                        <span className={p.is_used ? 'badge-success' : 'badge-warning'}>
                                            <span className={`w-1.5 h-1.5 rounded-full ${p.is_used ? 'bg-success' : 'bg-warning'}`} />
                                            {p.is_used ? 'Использован' : 'Не использован'}
                                        </span>
                                    </td>
                                    <td>{p.upload_date?.split('T')[0]}</td>
                                    <td>
                                        {!p.is_used && user?.role === 'customs_staff' && (
                                            <button
                                                onClick={() => handleMarkUsed(p.track_number)}
                                                className="text-xs text-primary font-medium hover:text-primary-dark cursor-pointer"
                                            >
                                                Использовать
                                            </button>
                                        )}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}

                {/* Pagination */}
                <div className="flex items-center justify-between px-4 py-3 border-t border-border">
                    <span className="text-sm text-text-muted">Showing {parcels.length} of {total} entries (page {page})</span>
                    <div className="pagination">
                        <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}>Previous</button>
                        {Array.from({ length: Math.min(5, totalPages) }, (_, i) => i + 1).map((n) => (
                            <button key={n} onClick={() => setPage(n)} className={page === n ? 'active' : ''}>{n}</button>
                        ))}
                        <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages}>Next</button>
                    </div>
                </div>
            </div>
        </div>
    );
}
