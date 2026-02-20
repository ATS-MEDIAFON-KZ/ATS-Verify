import { useState } from 'react';
import { MapPin, Search, Truck, CheckCircle, AlertCircle, ExternalLink } from 'lucide-react';
import api from '../lib/api';

interface TrackingEvent {
    id: string;
    status_code: string;
    description: string;
    location: string;
    event_time: string;
    source: string;
}

interface TrackingResponse {
    track_number: string;
    events: TrackingEvent[] | null;
    provider: string;
    external_url?: string;
    parcel?: {
        track_number: string;
        marketplace: string;
        product_name: string;
    };
}

export default function TrackingPage() {
    const [trackNumber, setTrackNumber] = useState('');
    const [loading, setLoading] = useState(false);
    const [result, setResult] = useState<TrackingResponse | null>(null);
    const [error, setError] = useState('');

    const handleTrack = async () => {
        const trimmed = trackNumber.trim();
        if (!trimmed) return;
        setLoading(true);
        setError('');
        setResult(null);

        try {
            const { data } = await api.get<TrackingResponse>(`/tracking/${encodeURIComponent(trimmed)}`);
            setResult(data);
        } catch (err: unknown) {
            const axiosErr = err as { response?: { status: number; data?: { error?: string } } };
            if (axiosErr.response?.status === 404) {
                setError('Трек-номер не найден ни в одной системе (Казпочта/СДЭК). Проверьте правильность номера.');
            } else {
                setError('Ошибка при отслеживании. Попробуйте позже.');
            }
        } finally {
            setLoading(false);
        }
    };

    const formatDate = (iso: string) => {
        if (!iso) return '';
        const d = new Date(iso);
        if (isNaN(d.getTime())) return iso;
        return d.toLocaleString('ru-RU', {
            year: 'numeric',
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
        });
    };

    return (
        <div>
            <div className="page-header">
                <h1 className="page-title">Отслеживание посылки</h1>
                <p className="page-subtitle">Проверьте статус доставки через Казпочту или СДЭК</p>
            </div>

            {/* Search */}
            <div className="card p-6 mb-6">
                <div className="flex gap-3">
                    <div className="relative flex-1">
                        <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
                        <input
                            type="text"
                            value={trackNumber}
                            onChange={(e) => setTrackNumber(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && handleTrack()}
                            className="input !pl-10"
                            placeholder="Введите трек-номер (например, LK721664764CN или 10207062728)"
                        />
                    </div>
                    <button onClick={handleTrack} disabled={loading || !trackNumber.trim()} className="btn-primary disabled:opacity-50 disabled:cursor-not-allowed">
                        <Truck size={16} />
                        {loading ? 'Поиск...' : 'Отследить'}
                    </button>
                </div>
            </div>

            {/* Error */}
            {error && (
                <div className="card p-6 mb-6 border-danger/20 bg-danger-light">
                    <div className="flex items-center gap-3 text-danger">
                        <AlertCircle size={20} />
                        <p className="text-sm font-medium">{error}</p>
                    </div>
                </div>
            )}

            {/* CDEK External Link (when server can't fetch directly) */}
            {result && result.external_url && (!result.events || result.events.length === 0) && (
                <div className="card p-6 mb-6">
                    <div className="flex items-center justify-between mb-4">
                        <h3 className="text-base font-semibold text-text-primary">Отслеживание через {result.provider}</h3>
                        <span className="badge-info">{result.provider}</span>
                    </div>
                    <p className="text-sm text-text-secondary mb-4">
                        Для отслеживания посылок СДЭК откройте официальную страницу:
                    </p>
                    <a
                        href={result.external_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="btn-primary inline-flex items-center gap-2"
                    >
                        <ExternalLink size={16} />
                        Отследить на сайте СДЭК
                    </a>
                </div>
            )}

            {/* Timeline */}
            {result && result.events && result.events.length > 0 && (
                <div className="card p-6">
                    <div className="flex items-center justify-between mb-6">
                        <h3 className="text-base font-semibold text-text-primary">История доставки</h3>
                        <span className="badge-info">{result.provider}</span>
                    </div>

                    <div className="space-y-0">
                        {result.events!.map((event, i) => (
                            <div key={event.id || i} className="flex gap-4">
                                {/* Timeline line */}
                                <div className="flex flex-col items-center">
                                    <div className={`w-8 h-8 rounded-full flex items-center justify-center shrink-0 ${i === 0 ? 'bg-success text-white' : 'bg-primary-light text-primary'
                                        }`}>
                                        {i === 0 ? <CheckCircle size={16} /> : <MapPin size={14} />}
                                    </div>
                                    {i < result.events!.length - 1 && (
                                        <div className="w-px h-12 bg-border" />
                                    )}
                                </div>

                                {/* Content */}
                                <div className={`pb-6 ${i < result.events.length - 1 ? '' : 'pb-0'}`}>
                                    <p className="text-sm font-medium text-text-primary">{event.description}</p>
                                    <p className="text-xs text-text-secondary mt-0.5">{event.location}</p>
                                    <p className="text-xs text-text-muted mt-0.5">{formatDate(event.event_time)}</p>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
}
