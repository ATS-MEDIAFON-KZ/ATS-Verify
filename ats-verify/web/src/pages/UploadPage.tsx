import { useState, useRef, type DragEvent } from 'react';
import { Upload, FileText, CheckCircle, X } from 'lucide-react';
import api from '../lib/api';
import { useAuth } from '../hooks/useAuth';

interface UploadResult {
    inserted: number;
    updated: number;
    skipped: number;
    errors: string[];
}

export default function UploadPage() {
    const { user } = useAuth();
    const [file, setFile] = useState<File | null>(null);
    const [marketplace, setMarketplace] = useState('');
    const [uploading, setUploading] = useState(false);
    const [progress, setProgress] = useState(0);
    const [result, setResult] = useState<UploadResult | null>(null);
    const [error, setError] = useState('');
    const [dragOver, setDragOver] = useState(false);
    const inputRef = useRef<HTMLInputElement>(null);

    const handleDrop = (e: DragEvent) => {
        e.preventDefault();
        setDragOver(false);
        const f = e.dataTransfer.files[0];
        if (f?.name.endsWith('.csv')) setFile(f);
    };

    const handleUpload = async () => {
        if (!file) return;
        setUploading(true);
        setError('');
        setResult(null);
        setProgress(0);

        try {
            const formData = new FormData();
            formData.append('file', file);
            if (marketplace) {
                formData.append('marketplace', marketplace);
            }
            const { data } = await api.post('/parcels/upload', formData, {
                headers: { 'Content-Type': 'multipart/form-data' },
                onUploadProgress: (progressEvent) => {
                    if (progressEvent.total) {
                        const percent = Math.round((progressEvent.loaded * 100) / progressEvent.total);
                        setProgress(percent);
                    }
                }
            });
            setResult(data);
        } catch (err: unknown) {
            const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
            setError(msg || 'Ошибка загрузки файла. Убедитесь что бэкенд запущен.');
        } finally {
            setUploading(false);
        }
    };

    return (
        <div>
            <div className="page-header">
                <h1 className="page-title">Загрузка CSV</h1>
                <p className="page-subtitle">Загрузите файл с данными маркетплейса</p>
            </div>

            <div className="card p-6 mb-6">
                <h3 className="text-sm font-semibold text-text-primary mb-4 flex items-center gap-2">
                    <Upload size={16} className="text-primary" />
                    Файл данных
                </h3>

                {user?.role === 'admin' && (
                    <div className="mb-4">
                        <label className="block text-sm font-medium text-text-secondary mb-1">Маркетплейс (если отсутствует в файле)</label>
                        <select
                            value={marketplace}
                            onChange={(e) => setMarketplace(e.target.value)}
                            className="input w-full md:w-1/2"
                        >
                            <option value="">Выберите маркетплейс...</option>
                            <option value="kaspi">Kaspi</option>
                            <option value="ozon">Ozon</option>
                            <option value="wildberries">Wildberries</option>
                        </select>
                    </div>
                )}

                {/* Drop zone */}
                <div
                    onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
                    onDragLeave={() => setDragOver(false)}
                    onDrop={handleDrop}
                    onClick={() => inputRef.current?.click()}
                    className={`border-2 border-dashed rounded-xl p-10 text-center cursor-pointer transition-all ${dragOver ? 'border-primary bg-primary-50' : 'border-border hover:border-primary/40'
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
                            <Upload size={32} className="text-text-muted mx-auto mb-3" />
                            <p className="text-sm text-text-secondary">Перетащите CSV-файл сюда или <span className="text-primary font-medium">выберите файл</span></p>
                            <p className="text-xs text-text-muted mt-1">Поддерживается формат CSV (до 10 МБ)</p>
                        </div>
                    )}
                </div>

                {uploading && (
                    <div className="mt-4">
                        <div className="flex justify-between text-xs text-text-muted mb-1">
                            <span>Загрузка файла...</span>
                            <span>{progress}%</span>
                        </div>
                        <div className="w-full bg-border rounded-full h-2 overflow-hidden">
                            <div
                                className="bg-primary h-2 rounded-full transition-all duration-300"
                                style={{ width: `${progress}%` }}
                            ></div>
                        </div>
                    </div>
                )}

                {/* Error */}
                {error && (
                    <div className="mt-4 bg-danger-light border border-danger/20 text-danger text-sm px-4 py-3 rounded-lg">
                        {error}
                    </div>
                )}

                {/* Upload button */}
                <div className="mt-4 flex justify-end">
                    <button
                        onClick={handleUpload}
                        disabled={!file || uploading}
                        className="btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        <Upload size={16} />
                        {uploading ? 'Загрузка...' : 'Загрузить файл'}
                    </button>
                </div>
            </div>

            {/* Result */}
            {result && (
                <div className="card p-6">
                    <h3 className="text-sm font-semibold text-text-primary mb-4 flex items-center gap-2">
                        <CheckCircle size={16} className="text-success" />
                        Результат загрузки
                    </h3>
                    <div className="grid grid-cols-3 gap-4">
                        <div className="bg-success-light rounded-lg p-4 text-center">
                            <p className="text-2xl font-bold text-green-700">{result.inserted}</p>
                            <p className="text-xs text-green-600 mt-1">Добавлено</p>
                        </div>
                        <div className="bg-info-light rounded-lg p-4 text-center">
                            <p className="text-2xl font-bold text-blue-700">{result.updated}</p>
                            <p className="text-xs text-blue-600 mt-1">Обновлено</p>
                        </div>
                        <div className="bg-warning-light rounded-lg p-4 text-center">
                            <p className="text-2xl font-bold text-amber-700">{result.skipped}</p>
                            <p className="text-xs text-amber-600 mt-1">Пропущено</p>
                        </div>
                    </div>
                    {result.errors && result.errors.length > 0 && (
                        <div className="mt-4 bg-danger-light border border-danger/20 text-danger text-sm px-4 py-3 rounded-lg">
                            <p className="font-medium mb-1">Ошибки при обработке:</p>
                            <ul className="list-disc list-inside text-xs">
                                {result.errors.slice(0, 10).map((e, i) => <li key={i}>{e}</li>)}
                            </ul>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
