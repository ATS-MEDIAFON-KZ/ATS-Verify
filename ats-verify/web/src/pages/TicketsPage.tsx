import { useState, useEffect, useCallback } from 'react';
import {
    DndContext,
    DragOverlay,
    PointerSensor,
    useSensor,
    useSensors,
    closestCorners,
} from '@dnd-kit/core';
import type { DragEndEvent, DragStartEvent } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy, useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import {
    Plus,
    AlertCircle,
    ArrowRightCircle,
    CheckCircle2,
    MessageSquare,
    Paperclip,
    X,
    User,
} from 'lucide-react';
import api from '../lib/api';
import { useAuth } from '../hooks/useAuth';
import type { SupportTicket, TicketStatus, TicketPriority } from '../types';

const COLUMNS: { id: TicketStatus; label: string; color: string; dotClass: string; icon: React.ReactNode }[] = [
    { id: 'to_do', label: 'К выполнению', color: 'bg-blue-50 border-blue-200', dotClass: 'bg-blue-500', icon: <AlertCircle size={16} className="text-blue-500" /> },
    { id: 'in_progress', label: 'В работе', color: 'bg-amber-50 border-amber-200', dotClass: 'bg-amber-500', icon: <ArrowRightCircle size={16} className="text-amber-500" /> },
    { id: 'completed', label: 'Выполнено', color: 'bg-green-50 border-green-200', dotClass: 'bg-green-500', icon: <CheckCircle2 size={16} className="text-green-500" /> },
];

const PRIORITY_MAP: Record<TicketPriority, { label: string; class: string }> = {
    high: { label: 'Высокий', class: 'badge-danger' },
    medium: { label: 'Средний', class: 'badge-warning' },
    low: { label: 'Низкий', class: 'badge-neutral' },
};

// --- Sortable Ticket Card ---
function TicketCard({ ticket, onClick }: { ticket: SupportTicket; onClick: () => void }) {
    const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
        id: ticket.id,
        data: { ticket },
    });

    const style = {
        transform: CSS.Transform.toString(transform),
        transition,
        opacity: isDragging ? 0.4 : 1,
    };

    const p = PRIORITY_MAP[ticket.priority];

    return (
        <div
            ref={setNodeRef}
            style={style}
            {...attributes}
            {...listeners}
            onClick={onClick}
            className="kanban-card group"
        >
            <div className="flex items-center justify-between mb-2">
                <span className={p.class}>{p.label}</span>
                <span className="text-[11px] text-text-muted font-mono">#{ticket.support_ticket_id}</span>
            </div>
            <h4 className="text-sm font-semibold text-text-primary mb-1 line-clamp-2">{ticket.rejection_reason}</h4>
            <p className="text-xs text-text-muted mb-3 truncate">{ticket.full_name} · ИИН {ticket.iin}</p>
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 text-text-muted">
                    {ticket.attachments?.length > 0 && (
                        <span className="flex items-center gap-0.5 text-[11px]"><Paperclip size={12} />{ticket.attachments.length}</span>
                    )}
                    {(ticket.support_comment || ticket.customs_comment) && (
                        <span className="flex items-center gap-0.5 text-[11px]"><MessageSquare size={12} /></span>
                    )}
                </div>
                {ticket.assigned_to ? (
                    <div className="w-6 h-6 rounded-full bg-primary-light flex items-center justify-center">
                        <User size={12} className="text-primary" />
                    </div>
                ) : (
                    <div className="w-6 h-6 rounded-full bg-bg-muted flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
                        <Plus size={12} className="text-text-muted" />
                    </div>
                )}
            </div>
        </div>
    );
}

// --- Overlay Card (while dragging) ---
function DragOverlayCard({ ticket }: { ticket: SupportTicket }) {
    const p = PRIORITY_MAP[ticket.priority];
    return (
        <div className="kanban-card kanban-card-dragging">
            <div className="flex items-center justify-between mb-2">
                <span className={p.class}>{p.label}</span>
                <span className="text-[11px] text-text-muted font-mono">#{ticket.support_ticket_id}</span>
            </div>
            <h4 className="text-sm font-semibold text-text-primary mb-1">{ticket.rejection_reason}</h4>
            <p className="text-xs text-text-muted">{ticket.full_name}</p>
        </div>
    );
}

// --- Create Ticket Modal ---
function CreateTicketModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
    const [form, setForm] = useState({
        iin: '', full_name: '', support_ticket_id: '', application_number: '',
        document_number: '', rejection_reason: '', support_comment: '', priority: 'medium',
    });
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);
        setError('');
        try {
            await api.post('/tickets', form);
            onCreated();
        } catch (err: unknown) {
            const msg = (err as { response?: { data?: { message?: string } } })?.response?.data?.message || 'Ошибка создания';
            setError(msg);
        } finally {
            setLoading(false);
        }
    };

    const update = (key: string, value: string) => setForm((f) => ({ ...f, [key]: value }));

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" onClick={onClose}>
            <div className="card p-6 w-full max-w-lg max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
                <div className="flex items-center justify-between mb-5">
                    <h2 className="text-lg font-bold text-text-primary">Новый тикет</h2>
                    <button onClick={onClose} className="text-text-muted hover:text-text-primary cursor-pointer"><X size={18} /></button>
                </div>
                {error && <div className="badge-danger text-sm mb-4 p-2 rounded-lg">{error}</div>}
                <form onSubmit={handleSubmit} className="space-y-3">
                    <div className="grid grid-cols-2 gap-3">
                        <input className="input" placeholder="ИИН *" value={form.iin} onChange={(e) => update('iin', e.target.value)} required />
                        <input className="input" placeholder="ФИО *" value={form.full_name} onChange={(e) => update('full_name', e.target.value)} required />
                    </div>
                    <div className="grid grid-cols-2 gap-3">
                        <input className="input" placeholder="ID тикета *" value={form.support_ticket_id} onChange={(e) => update('support_ticket_id', e.target.value)} required />
                        <input className="input" placeholder="Номер заявки *" value={form.application_number} onChange={(e) => update('application_number', e.target.value)} required />
                    </div>
                    <input className="input" placeholder="Номер документа *" value={form.document_number} onChange={(e) => update('document_number', e.target.value)} required />
                    <textarea className="input min-h-[80px] resize-none" placeholder="Причина отказа *" value={form.rejection_reason} onChange={(e) => update('rejection_reason', e.target.value)} required />
                    <textarea className="input min-h-[60px] resize-none" placeholder="Комментарий поддержки" value={form.support_comment} onChange={(e) => update('support_comment', e.target.value)} />
                    <select className="input" value={form.priority} onChange={(e) => update('priority', e.target.value)}>
                        <option value="low">Низкий приоритет</option>
                        <option value="medium">Средний приоритет</option>
                        <option value="high">Высокий приоритет</option>
                    </select>
                    <button type="submit" disabled={loading} className="btn-primary w-full justify-center">
                        {loading ? 'Создание...' : 'Создать тикет'}
                    </button>
                </form>
            </div>
        </div>
    );
}

// --- Ticket Detail Panel ---
function TicketDetailPanel({ ticket, onClose, onUpdated }: { ticket: SupportTicket; onClose: () => void; onUpdated: () => void }) {
    const { user } = useAuth();
    const [comment, setComment] = useState('');
    const isCustoms = user?.role === 'customs_staff' || user?.role === 'admin';

    const handleAddComment = async () => {
        if (!comment.trim()) return;
        const field = isCustoms ? 'customs_comment' : 'support_comment';
        try {
            await api.patch(`/tickets/${ticket.id}/comment`, { field, value: comment });
            setComment('');
            onUpdated();
        } catch { /* ignore */ }
    };

    const p = PRIORITY_MAP[ticket.priority];
    const statusCol = COLUMNS.find((c) => c.id === ticket.status);

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm" onClick={onClose}>
            <div className="card p-6 w-full max-w-xl max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
                <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center gap-2">
                        <span className={p.class}>{p.label}</span>
                        {statusCol && <span className="badge-info">{statusCol.label}</span>}
                        <span className="text-xs text-text-muted font-mono">#{ticket.support_ticket_id}</span>
                    </div>
                    <button onClick={onClose} className="text-text-muted hover:text-text-primary cursor-pointer"><X size={18} /></button>
                </div>
                <h2 className="text-lg font-bold text-text-primary mb-1">{ticket.rejection_reason}</h2>
                <p className="text-sm text-text-muted mb-5">{ticket.full_name} · ИИН {ticket.iin}</p>

                <div className="grid grid-cols-2 gap-3 mb-5">
                    <div className="p-3 bg-bg-muted rounded-lg">
                        <p className="text-[11px] text-text-muted uppercase tracking-wider mb-0.5">Заявка</p>
                        <p className="text-sm font-medium text-text-primary">{ticket.application_number}</p>
                    </div>
                    <div className="p-3 bg-bg-muted rounded-lg">
                        <p className="text-[11px] text-text-muted uppercase tracking-wider mb-0.5">Документ</p>
                        <p className="text-sm font-medium text-text-primary">{ticket.document_number}</p>
                    </div>
                </div>

                {ticket.support_comment && (
                    <div className="mb-3 p-3 rounded-lg border border-border">
                        <p className="text-[11px] text-text-muted uppercase tracking-wider mb-1">Комментарий поддержки</p>
                        <p className="text-sm text-text-secondary">{ticket.support_comment}</p>
                    </div>
                )}

                {ticket.customs_comment && (
                    <div className="mb-3 p-3 rounded-lg border border-border">
                        <p className="text-[11px] text-text-muted uppercase tracking-wider mb-1">Комментарий таможни</p>
                        <p className="text-sm text-text-secondary">{ticket.customs_comment}</p>
                    </div>
                )}

                <div className="mt-4 flex gap-2">
                    <input
                        className="input flex-1"
                        placeholder={isCustoms ? 'Комментарий таможни...' : 'Комментарий поддержки...'}
                        value={comment}
                        onChange={(e) => setComment(e.target.value)}
                        onKeyDown={(e) => e.key === 'Enter' && handleAddComment()}
                    />
                    <button onClick={handleAddComment} className="btn-primary">
                        <MessageSquare size={14} />
                    </button>
                </div>
            </div>
        </div>
    );
}

// --- Main Kanban Page ---
export default function TicketsPage() {
    const { user } = useAuth();
    const [tickets, setTickets] = useState<SupportTicket[]>([]);
    const [loading, setLoading] = useState(true);
    const [showCreate, setShowCreate] = useState(false);
    const [selectedTicket, setSelectedTicket] = useState<SupportTicket | null>(null);
    const [activeId, setActiveId] = useState<string | null>(null);

    const sensors = useSensors(
        useSensor(PointerSensor, { activationConstraint: { distance: 8 } })
    );

    const fetchTickets = useCallback(async () => {
        try {
            const res = await api.get('/tickets');
            setTickets(res.data || []);
        } catch {
            setTickets([]);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => { fetchTickets(); }, [fetchTickets]);

    const ticketsByStatus = (status: TicketStatus) => tickets.filter((t) => t.status === status);

    const handleDragStart = (event: DragStartEvent) => {
        setActiveId(event.active.id as string);
    };

    const handleDragEnd = async (event: DragEndEvent) => {
        setActiveId(null);
        const { active, over } = event;
        if (!over) return;

        const ticketId = active.id as string;
        const newStatus = over.id as TicketStatus;

        const ticket = tickets.find((t) => t.id === ticketId);
        if (!ticket || ticket.status === newStatus) return;

        // Optimistic update.
        setTickets((prev) => prev.map((t) => t.id === ticketId ? { ...t, status: newStatus } : t));

        try {
            await api.patch(`/tickets/${ticketId}/status`, { status: newStatus });
        } catch {
            // Rollback on error.
            fetchTickets();
        }
    };

    const activeTicket = activeId ? tickets.find((t) => t.id === activeId) : null;
    const canCreate = user?.role === 'ats_staff' || user?.role === 'admin';

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <div className="w-8 h-8 border-3 border-primary border-t-transparent rounded-full animate-spin" />
            </div>
        );
    }

    return (
        <div>
            {/* Header */}
            <div className="flex items-center justify-between mb-6">
                <div className="page-header mb-0">
                    <h1 className="page-title">Тикеты поддержки</h1>
                    <p className="page-subtitle">Канбан-доска для обработки обращений</p>
                </div>
                <div className="flex items-center gap-3">
                    {/* Stats */}
                    <div className="hidden lg:flex items-center gap-4 mr-2">
                        {COLUMNS.map((col) => (
                            <div key={col.id} className="flex items-center gap-1.5 text-sm text-text-muted">
                                <span className={`w-2 h-2 rounded-full ${col.dotClass}`} />
                                <span>{ticketsByStatus(col.id).length}</span>
                            </div>
                        ))}
                    </div>
                    {canCreate && (
                        <button onClick={() => setShowCreate(true)} className="btn-primary">
                            <Plus size={16} /> Новый тикет
                        </button>
                    )}
                </div>
            </div>

            {/* Kanban Board */}
            <DndContext
                sensors={sensors}
                collisionDetection={closestCorners}
                onDragStart={handleDragStart}
                onDragEnd={handleDragEnd}
            >
                <div className="kanban-board">
                    {COLUMNS.map((col) => {
                        const columnTickets = ticketsByStatus(col.id);
                        return (
                            <SortableContext
                                key={col.id}
                                id={col.id}
                                items={columnTickets.map((t) => t.id)}
                                strategy={verticalListSortingStrategy}
                            >
                                <div className="kanban-column">
                                    <div className={`kanban-column-header ${col.color}`}>
                                        <div className="flex items-center gap-2">
                                            {col.icon}
                                            <span className="text-sm font-semibold text-text-primary">{col.label}</span>
                                        </div>
                                        <span className="kanban-count">{columnTickets.length}</span>
                                    </div>
                                    <div
                                        className="kanban-column-body"
                                        data-column-id={col.id}
                                    >
                                        {columnTickets.length === 0 ? (
                                            <div className="kanban-empty">
                                                <p className="text-xs text-text-muted">Нет тикетов</p>
                                            </div>
                                        ) : (
                                            columnTickets.map((ticket) => (
                                                <TicketCard
                                                    key={ticket.id}
                                                    ticket={ticket}
                                                    onClick={() => setSelectedTicket(ticket)}
                                                />
                                            ))
                                        )}
                                    </div>
                                </div>
                            </SortableContext>
                        );
                    })}
                </div>

                <DragOverlay>
                    {activeTicket ? <DragOverlayCard ticket={activeTicket} /> : null}
                </DragOverlay>
            </DndContext>

            {/* Modals */}
            {showCreate && (
                <CreateTicketModal
                    onClose={() => setShowCreate(false)}
                    onCreated={() => { setShowCreate(false); fetchTickets(); }}
                />
            )}
            {selectedTicket && (
                <TicketDetailPanel
                    ticket={selectedTicket}
                    onClose={() => setSelectedTicket(null)}
                    onUpdated={() => { setSelectedTicket(null); fetchTickets(); }}
                />
            )}
        </div>
    );
}
