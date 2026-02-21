import { useState, useEffect, useCallback } from 'react';
import { ShieldAlert, Plus, X, Search, Trash2 } from 'lucide-react';
import api from '../lib/api';
import type { RiskProfile, RiskLevel } from '../types';

const LEVEL_CONFIG: Record<RiskLevel, { badge: string; label: string }> = {
    red: { badge: 'badge-danger', label: 'Высокий' },
    yellow: { badge: 'badge-warning', label: 'Средний' },
    green: { badge: 'badge-success', label: 'Низкий' },
};

export default function RisksPage() {
    const [risks, setRisks] = useState<RiskProfile[]>([]);
    const [searchQuery, setSearchQuery] = useState('');
    const [showModal, setShowModal] = useState(false);
    const [newIin, setNewIin] = useState('');
    const [newLevel, setNewLevel] = useState<RiskLevel>('yellow');
    const [newReason, setNewReason] = useState('');
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState('');

    const fetchRisks = useCallback(async () => {
        try {
            const { data } = await api.get('/risks');
            setRisks(data.profiles || []);
        } catch {
            setRisks([]);
        }
    }, []);

    useEffect(() => { fetchRisks(); }, [fetchRisks]);

    const handleSave = async () => {
        if (!newIin || !newLevel) return;
        setSaving(true);
        setError('');
        try {
            await api.post('/risks', {
                iin_bin: newIin,
                risk_level: newLevel,
                comment: newReason, // API expects 'comment'
            });
            setShowModal(false);
            setNewIin(''); setNewLevel('yellow'); setNewReason('');
            fetchRisks();
        } catch (err: unknown) {
            const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
            setError(msg || 'Ошибка сохранения');
        } finally {
            setSaving(false);
        }
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Удалить профиль риска?')) return;
        try {
            await api.delete(`/risks/${id}`);
            fetchRisks();
        } catch {
            alert('Ошибка удаления');
        }
    };

    const filtered = risks.filter((r) =>
        r.iin_bin.toLowerCase().includes(searchQuery.toLowerCase()) ||
        r.comment?.toLowerCase().includes(searchQuery.toLowerCase())
    );

    return (
        <div>
            {/* Header */}
            <div className="flex items-center justify-between mb-6">
                <div>
                    <h1 className="page-title">Профили рисков</h1>
                    <p className="page-subtitle">Управление рисковыми профилями ИИН/БИН</p>
                </div>
                <button onClick={() => setShowModal(true)} className="btn-primary">
                    <Plus size={16} />
                    Добавить ИИН/БИН
                </button>
            </div>

            {/* Search */}
            <div className="card p-4 mb-4">
                <div className="relative max-w-sm">
                    <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
                    <input
                        type="text"
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="input !pl-10"
                        placeholder="Поиск по ИИН/БИН или причине..."
                    />
                </div>
            </div>

            {/* Table */}
            <div className="card overflow-hidden">
                {filtered.length === 0 ? (
                    <div className="p-8 text-center text-text-muted">Нет профилей рисков.</div>
                ) : (
                    <table className="data-table">
                        <thead>
                            <tr>
                                <th>ИИН / БИН</th>
                                <th>Уровень риска</th>
                                <th>Причина</th>
                                <th>Дата добавления</th>
                                <th>Действия</th>
                            </tr>
                        </thead>
                        <tbody>
                            {filtered.map((profile) => (
                                <tr key={profile.id}>
                                    <td className="font-mono font-medium text-text-primary">{profile.iin_bin}</td>
                                    <td>
                                        <span className={LEVEL_CONFIG[profile.risk_level]?.badge || 'badge-info'}>
                                            {LEVEL_CONFIG[profile.risk_level]?.label || profile.risk_level}
                                        </span>
                                    </td>
                                    <td>{profile.comment}</td>
                                    <td>{new Date(profile.created_at).toLocaleDateString()}</td>
                                    <td>
                                        <button onClick={() => handleDelete(profile.id)} className="text-xs text-danger font-medium hover:text-red-700 cursor-pointer flex items-center gap-1">
                                            <Trash2 size={12} />
                                            Удалить
                                        </button>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}

                <div className="px-4 py-3 border-t border-border">
                    <span className="text-sm text-text-muted">Showing {filtered.length} of {risks.length} entries</span>
                </div>
            </div>

            {/* Modal */}
            {showModal && (
                <div className="fixed inset-0 z-50 flex items-center justify-center">
                    <div className="absolute inset-0 bg-black/30" onClick={() => setShowModal(false)} />
                    <div className="relative bg-bg-white rounded-xl p-6 w-full max-w-md shadow-xl border border-border">
                        <div className="flex items-center justify-between mb-5">
                            <h3 className="text-lg font-semibold text-text-primary flex items-center gap-2">
                                <ShieldAlert size={18} className="text-primary" />
                                Добавить профиль риска
                            </h3>
                            <button onClick={() => setShowModal(false)} className="text-text-muted hover:text-text-primary cursor-pointer">
                                <X size={18} />
                            </button>
                        </div>

                        {error && (
                            <div className="mb-4 bg-danger-light border border-danger/20 text-danger text-sm px-4 py-3 rounded-lg">
                                {error}
                            </div>
                        )}

                        <div className="space-y-4">
                            <div>
                                <label className="text-sm font-medium text-text-primary block mb-1.5">ИИН / БИН</label>
                                <input value={newIin} onChange={(e) => setNewIin(e.target.value.replace(/\D/g, '').slice(0, 12))} className="input" placeholder="12-значный номер" />
                            </div>

                            <div>
                                <label className="text-sm font-medium text-text-primary block mb-1.5">Уровень риска</label>
                                <div className="flex gap-2">
                                    {(['red', 'yellow', 'green'] as const).map((lvl) => (
                                        <button
                                            key={lvl}
                                            onClick={() => setNewLevel(lvl)}
                                            className={`px-4 py-2 rounded-lg text-sm font-medium border transition-all cursor-pointer ${newLevel === lvl
                                                ? lvl === 'red' ? 'bg-danger text-white border-danger'
                                                    : lvl === 'yellow' ? 'bg-warning text-white border-warning'
                                                        : 'bg-success text-white border-success'
                                                : 'bg-bg-white border-border text-text-secondary hover:border-primary/30'
                                                }`}
                                        >
                                            {LEVEL_CONFIG[lvl].label}
                                        </button>
                                    ))}
                                </div>
                            </div>

                            <div>
                                <label className="text-sm font-medium text-text-primary block mb-1.5">Причина</label>
                                <textarea value={newReason} onChange={(e) => setNewReason(e.target.value)} className="input h-20 resize-none" placeholder="Опишите причину..." />
                            </div>
                        </div>

                        <div className="flex justify-end gap-2 mt-6">
                            <button onClick={() => setShowModal(false)} className="btn-secondary">Отмена</button>
                            <button onClick={handleSave} disabled={saving || !newIin} className="btn-primary disabled:opacity-50">
                                <Plus size={16} />
                                {saving ? 'Сохранение...' : 'Добавить'}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
