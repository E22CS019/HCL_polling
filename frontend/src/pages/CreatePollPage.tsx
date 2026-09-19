import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'react-hot-toast'
import { PlusCircle, Trash2, GripVertical } from 'lucide-react'
import { createPoll } from '../api/polls'
import Input from '../components/ui/Input'
import Button from '../components/ui/Button'

export default function CreatePollPage() {
  const navigate = useNavigate()

  const [title,       setTitle]       = useState('')
  const [description, setDescription] = useState('')
  const [options,     setOptions]     = useState(['', ''])
  const [multiChoice, setMultiChoice] = useState(false)
  const [endsAt,      setEndsAt]      = useState('')
  const [loading,     setLoading]     = useState(false)
  const [errors,      setErrors]      = useState<Record<string, string>>({})

  function addOption() {
    if (options.length < 10) setOptions(prev => [...prev, ''])
  }

  function removeOption(i: number) {
    if (options.length <= 2) return
    setOptions(prev => prev.filter((_, idx) => idx !== i))
  }

  function updateOption(i: number, val: string) {
    setOptions(prev => prev.map((o, idx) => idx === i ? val : o))
  }

  function validate(): boolean {
    const errs: Record<string, string> = {}
    if (!title.trim())               errs.title   = 'Title is required.'
    if (title.trim().length < 3)     errs.title   = 'Title must be at least 3 characters.'
    const filled = options.filter(o => o.trim())
    if (filled.length < 2)           errs.options = 'At least 2 non-empty options are required.'
    if (new Set(filled).size !== filled.length)
                                     errs.options = 'Options must be unique.'
    if (endsAt && new Date(endsAt) <= new Date())
                                     errs.endsAt  = 'End date must be in the future.'
    setErrors(errs)
    return Object.keys(errs).length === 0
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!validate()) return
    setLoading(true)
    try {
      const poll = await createPoll({
        title:       title.trim(),
        description: description.trim(),
        options:     options.filter(o => o.trim()),
        multi_choice: multiChoice,
        ends_at:     endsAt ? new Date(endsAt).toISOString() : null,
      })
      toast.success('Poll created!')
      navigate(`/poll/${poll.id}`)
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })
        ?.response?.data?.error ?? 'Failed to create poll.'
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-2xl mx-auto px-4 py-12">
      <h1 className="text-3xl font-bold mb-2">Create a Poll</h1>
      <p className="text-slate-400 mb-8 text-sm">Fill in the details, share the link, watch the votes pour in.</p>

      <form onSubmit={handleSubmit} className="flex flex-col gap-6">
        {/* Title */}
        <div className="card flex flex-col gap-4">
          <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wider">Question</h2>
          <Input
            label="Title"
            value={title}
            onChange={e => setTitle(e.target.value)}
            placeholder="What's your favourite programming language?"
            maxLength={200}
            error={errors.title}
            required
          />
          <div className="flex flex-col gap-1.5">
            <label className="text-sm text-slate-300 font-medium">Description <span className="text-slate-600">(optional)</span></label>
            <textarea
              value={description}
              onChange={e => setDescription(e.target.value)}
              placeholder="Any extra context for voters…"
              maxLength={1000}
              rows={3}
              className="w-full px-4 py-2.5 rounded-xl bg-white/5 border border-white/10 hover:border-white/20 text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-brand-500/50 transition-all resize-none"
            />
          </div>
        </div>

        {/* Options */}
        <div className="card flex flex-col gap-3">
          <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wider">Options</h2>
          {errors.options && (
            <p className="text-xs text-red-400">{errors.options}</p>
          )}
          <div className="flex flex-col gap-2">
            {options.map((opt, i) => (
              <div key={i} className="flex items-center gap-2">
                <GripVertical size={16} className="text-slate-600 shrink-0" />
                <input
                  type="text"
                  value={opt}
                  onChange={e => updateOption(i, e.target.value)}
                  placeholder={`Option ${i + 1}`}
                  maxLength={200}
                  className="flex-1 px-4 py-2.5 rounded-xl bg-white/5 border border-white/10 hover:border-white/20 text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-brand-500/50 transition-all text-sm"
                />
                <button
                  type="button"
                  onClick={() => removeOption(i)}
                  disabled={options.length <= 2}
                  className="p-2 rounded-lg text-slate-500 hover:text-red-400 hover:bg-red-500/10 transition-all disabled:opacity-30 disabled:cursor-not-allowed"
                  aria-label="Remove option"
                >
                  <Trash2 size={15} />
                </button>
              </div>
            ))}
          </div>
          {options.length < 10 && (
            <button
              type="button"
              onClick={addOption}
              className="flex items-center gap-1.5 text-sm text-brand-400 hover:text-brand-300 transition-colors py-1"
            >
              <PlusCircle size={15} />
              Add option
            </button>
          )}
        </div>

        {/* Settings */}
        <div className="card flex flex-col gap-4">
          <h2 className="text-sm font-semibold text-slate-300 uppercase tracking-wider">Settings</h2>

          <label className="flex items-start gap-3 cursor-pointer group">
            <div className="relative mt-0.5">
              <input
                type="checkbox"
                checked={multiChoice}
                onChange={e => setMultiChoice(e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-10 h-6 bg-white/10 peer-checked:bg-brand-600 rounded-full transition-colors border border-white/20" />
              <div className="absolute top-1 left-1 w-4 h-4 bg-white rounded-full transition-transform peer-checked:translate-x-4" />
            </div>
            <div>
              <p className="text-sm font-medium text-slate-200">Allow multiple choices</p>
              <p className="text-xs text-slate-500 mt-0.5">Voters can select more than one option</p>
            </div>
          </label>

          <div className="flex flex-col gap-1.5">
            <label className="text-sm text-slate-300 font-medium">
              Close poll at <span className="text-slate-600">(optional)</span>
            </label>
            <input
              type="datetime-local"
              value={endsAt}
              onChange={e => setEndsAt(e.target.value)}
              min={new Date(Date.now() + 60_000).toISOString().slice(0, 16)}
              className="w-full px-4 py-2.5 rounded-xl bg-white/5 border border-white/10 hover:border-white/20 text-white focus:outline-none focus:ring-2 focus:ring-brand-500/50 transition-all text-sm"
            />
            {errors.endsAt && <p className="text-xs text-red-400">{errors.endsAt}</p>}
          </div>
        </div>

        <Button type="submit" loading={loading} size="lg" className="self-end px-10">
          Create Poll
        </Button>
      </form>
    </div>
  )
}
