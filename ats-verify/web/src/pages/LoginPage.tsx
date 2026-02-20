import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import api from '../lib/api';
import { Lock, User, CheckCircle2 } from 'lucide-react';

export default function LoginPage() {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const { login } = useAuth();
    const navigate = useNavigate();

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        setError('');
        setLoading(true);
        try {
            const { data } = await api.post('/auth/login', { username, password });
            login({ id: data.user.id, username: data.user.username, role: data.user.role, marketplace_prefix: data.user.marketplace_prefix, token: data.token });
            navigate('/');
        } catch (err: unknown) {
            const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
            setError(msg || 'Ошибка авторизации');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="min-h-screen flex items-center justify-center bg-bg px-4">
            <div className="w-full max-w-md">
                {/* Logo */}
                <div className="text-center mb-8">
                    <div className="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-primary mb-4">
                        <CheckCircle2 size={28} className="text-white" />
                    </div>
                    <h1 className="text-3xl font-bold text-text-primary">ATS-Verify</h1>
                    <p className="text-text-muted mt-1 text-sm">Платформа верификации</p>
                </div>

                {/* Login Card */}
                <div className="card p-8">
                    <h2 className="text-lg font-semibold text-text-primary mb-6">Вход в систему</h2>

                    <form onSubmit={handleSubmit} className="space-y-4">
                        {error && (
                            <div className="bg-danger-light border border-danger/20 text-danger text-sm px-4 py-3 rounded-lg">
                                {error}
                            </div>
                        )}

                        <div>
                            <label className="text-sm font-medium text-text-primary block mb-1.5">Логин</label>
                            <div className="relative">
                                <User size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
                                <input
                                    type="text"
                                    value={username}
                                    onChange={(e) => setUsername(e.target.value)}
                                    className="input !pl-10"
                                    placeholder="Введите логин"
                                    required
                                />
                            </div>
                        </div>

                        <div>
                            <label className="text-sm font-medium text-text-primary block mb-1.5">Пароль</label>
                            <div className="relative">
                                <Lock size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
                                <input
                                    type="password"
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    className="input !pl-10"
                                    placeholder="Введите пароль"
                                    required
                                />
                            </div>
                        </div>

                        <button type="submit" disabled={loading} className="btn-primary w-full justify-center py-2.5">
                            {loading ? 'Вход...' : 'Войти в систему'}
                        </button>
                    </form>
                </div>

                <p className="text-center text-text-muted text-xs mt-6">
                    © 2026 ATS-Verify. Все права защищены.
                </p>
            </div>
        </div>
    );
}
