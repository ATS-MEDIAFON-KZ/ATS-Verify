import { useState, useRef, useEffect, type DragEvent } from 'react';
import { Upload, FileText, X, ShieldAlert, BarChart3, AlertTriangle, RefreshCw } from 'lucide-react';
import api from '../lib/api';
import { useAuth } from '../hooks/useAuth';

interface AnalyticsResult {
    inserted_rows: number;
    message: string;
}

interface DocumentReport {
    document_number: string;
    usage_count: number;
    last_used: string;
}

interface DocumentIINReport {
    document_number: string;
    iin_count: number;
    iins: string;
}

interface HighFrequencyIINReport {
    iin: string;
    usage_count: number;
    last_used: string;
}

interface FlipFlopReport {
    document_number: string;
    approved_count: number;
    rejected_count: number;
}

interface AnalyticsReports {
    document_reuse: DocumentReport[];
    document_iin_reuse: DocumentIINReport[];
    high_frequency_iin: HighFrequencyIINReport[];
    flip_flop: FlipFlopReport[];
}

export default function AnalyticsPage() {
    const { user } = useAuth();
    const [file, setFile] = useState<File | null>(null);
    const [uploading, setUploading] = useState(false);
    const [result, setResult] = useState<AnalyticsResult | null>(null);
    const [error, setError] = useState('');
    const [dragOver, setDragOver] = useState(false);
    const inputRef = useRef<HTMLInputElement>(null);

    const [reports, setReports] = useState<AnalyticsReports | null>(null);
    const [loadingReports, setLoadingReports] = useState(false);

    const handleDrop = (e: DragEvent) => {
        e.preventDefault();
        setDragOver(false);
        const f = e.dataTransfer.files[0];
        if (f?.name.endsWith('.csv')) setFile(f);
    };

    const fetchReports = async () => {
        setLoadingReports(true);
        try {
            const { data } = await api.get('/risks/reports');
            setReports(data);
        } catch (err: unknown) {
            console.error('Failed to fetch reports:', err);
        } finally {
            setLoadingReports(false);
        }
    };

    useEffect(() => {
        fetchReports();
    }, []);

    const handleUpload = async () => {
        if (!file) return;
        setUploading(true);
        setError('');
        setResult(null);

        try {
            const formData = new FormData();
            formData.append('file', file);
            const { data } = await api.post('/risks/analyze', formData, {
                headers: { 'Content-Type': 'multipart/form-data' },
            });
            setResult(data);
            fetchReports(); // Refresh reports after upload
        } catch (err: unknown) {
            const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
            setError(msg || 'Ошибка загрузки файла. Убедитесь что бэкенд запущен.');
        } finally {
            setUploading(false);
        }
    };

    return (
        <div className="space-y-6">
            <div className="page-header flex justify-between items-start">
                <div>
                    <h1 className="page-title">Аналитика и Отчеты Risk Engine</h1>
                    <p className="page-subtitle">Загрузка сырых данных рисков и аналитические отчеты</p>
                </div>
                <button onClick={fetchReports} className="btn-secondary" title="Обновить отчеты">
                    <RefreshCw size={16} className={loadingReports ? 'animate-spin' : ''} />
                </button>
            </div>

            {/* Upload Section - Available only to Admin */}
            {user?.role === 'admin' && (
                <div className="card p-6">
                    <h3 className="text-sm font-semibold text-text-primary mb-4 flex items-center gap-2">
                        <BarChart3 size={16} className="text-primary" />
                        Импорт сырых данных (Risk CSV)
                    </h3>

                    <div
                        onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
                        onDragLeave={() => setDragOver(false)}
                        onDrop={handleDrop}
                        onClick={() => inputRef.current?.click()}
                        className={`border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-all ${dragOver ? 'border-primary bg-primary-50' : 'border-border hover:border-primary/40'
                            }`}
                    >
                        <input
                            ref={inputRef}
                            type="file"
                            accept=".csv"
                            onChange={(e) => e.target.files?.[0] && setFile(e.target.files[0])}
                            className="hidden"
                        />

                        {file ? (
                            <div className="flex items-center justify-center gap-3">
                                <FileText size={20} className="text-primary" />
                                <span className="text-sm font-medium text-text-primary">{file.name}</span>
                                <span className="text-xs text-text-muted">({(file.size / 1024).toFixed(1)} KB)</span>
                                <button onClick={(e) => { e.stopPropagation(); setFile(null); setResult(null); setError(''); }} className="text-text-muted hover:text-danger cursor-pointer">
                                    <X size={16} />
                                </button>
                            </div>
                        ) : (
                            <div>
                                <Upload size={28} className="text-text-muted mx-auto mb-3" />
                                <p className="text-sm text-text-secondary">Перетащите CSV-файл сюда или <span className="text-primary font-medium">выберите файл</span></p>
                                <p className="text-xs text-text-muted mt-1">Ожидаемые колонки: Date, Application Id, IIN/BIN, document, User, Organization, Status, Reject, Reason</p>
                            </div>
                        )}
                    </div>

                    {error && (
                        <div className="mt-4 bg-danger-light border border-danger/20 text-danger text-sm px-4 py-3 rounded-lg">
                            {error}
                        </div>
                    )}

                    {result && (
                        <div className="mt-4 bg-success-light border border-success/20 text-success text-sm px-4 py-3 rounded-lg flex items-center justify-between">
                            <span>{result.message}</span>
                            <span className="font-bold">Добавлено строк: {result.inserted_rows}</span>
                        </div>
                    )}

                    <div className="mt-4 flex justify-end">
                        <button
                            onClick={handleUpload}
                            disabled={!file || uploading}
                            className="btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            <ShieldAlert size={16} />
                            {uploading ? 'Анализ...' : 'Запустить Risk-Анализ'}
                        </button>
                    </div>
                </div>
            )}

            {/* Reports Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">

                {/* Repor 1: Document Reuse */}
                <div className="card overflow-hidden">
                    <div className="px-5 py-4 border-b border-border flex items-center gap-2 bg-red-50/30">
                        <AlertTriangle size={16} className="text-red-500" />
                        <h3 className="font-semibold text-text-primary text-sm">Документ использован несколько раз</h3>
                    </div>
                    <div className="p-0 overflow-x-auto max-h-[300px] overflow-y-auto">
                        <table className="w-full text-left text-sm whitespace-nowrap">
                            <thead className="bg-bg-muted text-xs text-text-secondary sticky top-0">
                                <tr>
                                    <th className="px-5 py-3 font-medium">Документ</th>
                                    <th className="px-5 py-3 font-medium text-center">Кол-во</th>
                                    <th className="px-5 py-3 font-medium text-right">Последний раз</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-border">
                                {reports?.document_reuse?.length ? (
                                    reports.document_reuse.map((r, i) => (
                                        <tr key={i} className="hover:bg-bg-hover">
                                            <td className="px-5 py-3 font-mono text-text-primary">{r.document_number}</td>
                                            <td className="px-5 py-3 text-center">
                                                <span className="badge-danger">{r.usage_count}</span>
                                            </td>
                                            <td className="px-5 py-3 text-right text-text-muted">{r.last_used?.split('T')[0]}</td>
                                        </tr>
                                    ))
                                ) : (
                                    <tr><td colSpan={3} className="px-5 py-8 text-center text-text-muted">Нет данных</td></tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>

                {/* Report 2: Document IIN Reuse */}
                <div className="card overflow-hidden">
                    <div className="px-5 py-4 border-b border-border flex items-center gap-2 bg-red-50/30">
                        <AlertTriangle size={16} className="text-red-500" />
                        <h3 className="font-semibold text-text-primary text-sm">Один документ у разных ИИН/БИН</h3>
                    </div>
                    <div className="p-0 overflow-x-auto max-h-[300px] overflow-y-auto">
                        <table className="w-full text-left text-sm whitespace-nowrap">
                            <thead className="bg-bg-muted text-xs text-text-secondary sticky top-0">
                                <tr>
                                    <th className="px-5 py-3 font-medium">Документ</th>
                                    <th className="px-5 py-3 font-medium text-center">Разных ИИН</th>
                                    <th className="px-5 py-3 font-medium">Список ИИН</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-border">
                                {reports?.document_iin_reuse?.length ? (
                                    reports.document_iin_reuse.map((r, i) => (
                                        <tr key={i} className="hover:bg-bg-hover">
                                            <td className="px-5 py-3 font-mono text-text-primary">{r.document_number}</td>
                                            <td className="px-5 py-3 text-center">
                                                <span className="badge-danger">{r.iin_count}</span>
                                            </td>
                                            <td className="px-5 py-3 font-mono text-xs text-text-secondary max-w-[150px] truncate" title={r.iins}>{r.iins}</td>
                                        </tr>
                                    ))
                                ) : (
                                    <tr><td colSpan={3} className="px-5 py-8 text-center text-text-muted">Нет данных</td></tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>

                {/* Report 3: High Frequency IIN */}
                <div className="card overflow-hidden">
                    <div className="px-5 py-4 border-b border-border flex items-center gap-2 bg-amber-50/30">
                        <AlertTriangle size={16} className="text-amber-500" />
                        <h3 className="font-semibold text-text-primary text-sm">Частая верификация ИИН/БИН (Аномалия)</h3>
                    </div>
                    <div className="p-0 overflow-x-auto max-h-[300px] overflow-y-auto">
                        <table className="w-full text-left text-sm whitespace-nowrap">
                            <thead className="bg-bg-muted text-xs text-text-secondary sticky top-0">
                                <tr>
                                    <th className="px-5 py-3 font-medium">ИИН/БИН</th>
                                    <th className="px-5 py-3 font-medium text-center">Заявок</th>
                                    <th className="px-5 py-3 font-medium text-right">Последний раз</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-border">
                                {reports?.high_frequency_iin?.length ? (
                                    reports.high_frequency_iin.map((r, i) => (
                                        <tr key={i} className="hover:bg-bg-hover">
                                            <td className="px-5 py-3 font-mono text-text-primary">{r.iin}</td>
                                            <td className="px-5 py-3 text-center">
                                                <span className="badge-warning">{r.usage_count}</span>
                                            </td>
                                            <td className="px-5 py-3 text-right text-text-muted">{r.last_used?.split('T')[0]}</td>
                                        </tr>
                                    ))
                                ) : (
                                    <tr><td colSpan={3} className="px-5 py-8 text-center text-text-muted">Нет данных</td></tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>

                {/* Report 4: Flip Flop */}
                <div className="card overflow-hidden">
                    <div className="px-5 py-4 border-b border-border flex items-center gap-2 bg-amber-50/30">
                        <AlertTriangle size={16} className="text-amber-500" />
                        <h3 className="font-semibold text-text-primary text-sm">Документ сначала принят, затем отклонён</h3>
                    </div>
                    <div className="p-0 overflow-x-auto max-h-[300px] overflow-y-auto">
                        <table className="w-full text-left text-sm whitespace-nowrap">
                            <thead className="bg-bg-muted text-xs text-text-secondary sticky top-0">
                                <tr>
                                    <th className="px-5 py-3 font-medium">Документ</th>
                                    <th className="px-5 py-3 font-medium text-center">Одобрено</th>
                                    <th className="px-5 py-3 font-medium text-center">Отклонено</th>
                                </tr>
                            </thead>
                            <tbody className="divide-y divide-border">
                                {reports?.flip_flop?.length ? (
                                    reports.flip_flop.map((r, i) => (
                                        <tr key={i} className="hover:bg-bg-hover">
                                            <td className="px-5 py-3 font-mono text-text-primary">{r.document_number}</td>
                                            <td className="px-5 py-3 text-center text-green-600 font-medium">{r.approved_count}</td>
                                            <td className="px-5 py-3 text-center text-red-600 font-medium">{r.rejected_count}</td>
                                        </tr>
                                    ))
                                ) : (
                                    <tr><td colSpan={3} className="px-5 py-8 text-center text-text-muted">Нет данных</td></tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>

            </div>
        </div>
    );
}
