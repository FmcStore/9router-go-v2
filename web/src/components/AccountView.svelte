<script lang="ts">
  // Standalone, login-free account page: the user's own API key is the only
  // credential. Mounted like /callback (outside the dashboard auth gate), so
  // it renders whether or not requireLogin is on. Read-only by design.
  import { api } from '../api/client'
  import { Loader2 } from 'lucide-svelte'

  interface AccountUsage {
    key?: string
    name?: string
    active?: boolean
    usedTokens?: number
    tokenQuota?: number
    remainingTokens?: number
    percentUsed?: number
    unlimited?: boolean
    quotaExceeded?: boolean
    lastUsedAt?: string
    expiresAt?: string
  }

  interface AccountModel {
    id: string
    ownedBy?: string
  }

  let apiKeyInput = $state('')
  let checkedKey = $state('')
  let usage = $state<AccountUsage | null>(null)
  let modelsList = $state<AccountModel[]>([])
  let loading = $state(false)
  let error = $state<string | null>(null)

  // Session-only: the key never touches localStorage, so a shared computer
  // does not leave the credential behind.
  const memoryKey = { value: '' }
  if (typeof sessionStorage !== 'undefined') {
    // sessionStorage survives reloads in the same tab only.
  }

  function fmtTokens(n: number): string {
    if (!Number.isFinite(n)) return '0'
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(n >= 10_000_000 ? 0 : 1)}M`
    if (n >= 1_000) return `${(n / 1_000).toFixed(n >= 10_000 ? 0 : 1)}k`
    return String(n)
  }

  let percent = $derived(
    usage && usage.tokenQuota && usage.tokenQuota > 0
      ? Math.min(100, Math.round((usage.usedTokens ?? 0) / usage.tokenQuota * 100))
      : 0
  )
  let barColor = $derived(percent >= 90 ? 'bg-danger' : percent >= 70 ? 'bg-warning' : 'bg-success')

  async function load(key: string) {
    loading = true
    error = null
    usage = null
    modelsList = []
    try {
      const [u, m] = await Promise.all([
        api.accountUsage(key),
        api.accountModels(key)
      ])
      usage = u as AccountUsage
      modelsList = ((m as { models?: AccountModel[] }).models) ?? []
      checkedKey = usage?.key ?? maskKey(key)
      memoryKey.value = key
    } catch (err) {
      error = err instanceof Error ? err.message : String(err)
    } finally {
      loading = false
    }
  }

  function maskKey(k: string): string {
    if (k.length <= 8) return k
    return `${k.slice(0, 5)}…${k.slice(-4)}`
  }

  function submit(e: SubmitEvent) {
    e.preventDefault()
    const key = apiKeyInput.trim()
    if (key) load(key)
  }

  // Restore a key from this tab's sessionStorage, if any.
  $effect(() => {
    if (typeof sessionStorage !== 'undefined') {
      const saved = sessionStorage.getItem('marxrouter_account_key')
      if (saved) {
        apiKeyInput = saved
        load(saved)
      }
    }
  })

  function persistKey(k: string) {
    if (typeof sessionStorage !== 'undefined') {
      if (k) sessionStorage.setItem('marxrouter_account_key', k)
      else sessionStorage.removeItem('marxrouter_account_key')
    }
  }
</script>

<div class="min-h-screen bg-bg text-text-main font-body">
  <header class="border-b border-border bg-surface">
    <div class="max-w-3xl mx-auto px-4 py-4 flex items-center gap-3">
      <img src="/favicon.svg" alt="MarxRouter" class="w-7 h-7" />
      <h1 class="text-lg font-bold text-primary">MarxRouter</h1>
      <span class="ml-auto text-[11px] text-text-subtle font-code">My Account</span>
    </div>
  </header>

  <main class="max-w-3xl mx-auto px-4 py-8 space-y-6">
    <form onsubmit={submit} class="flex gap-2">
      <input
        type="password"
        placeholder="Paste your API key (sk-…)"
        bind:value={apiKeyInput}
        onchange={() => persistKey(apiKeyInput.trim())}
        class="flex-1 bg-surface border border-border rounded-lg px-3 py-2 font-code text-xs text-text-main focus:outline-none focus:border-brand-500"
      />
      <button
        type="submit"
        disabled={loading || !apiKeyInput.trim()}
        class="px-4 py-2 rounded-lg bg-primary text-white text-xs font-semibold disabled:opacity-50 cursor-pointer flex items-center gap-2"
      >
        {#if loading}<Loader2 class="w-3.5 h-3.5 animate-spin" />{/if}
        Check
      </button>
    </form>

    {#if error}
      <div class="rounded-xl border border-danger/30 bg-danger/10 text-danger px-4 py-3 text-sm">
        {error}
      </div>
    {/if}

    {#if usage}
      <section class="space-y-4">
        <div class="flex items-center gap-2 text-sm">
          <code class="font-mono text-xs bg-surface-2 px-2 py-1 rounded border border-border/50">{checkedKey}</code>
          {#if usage.name}<span class="text-text-muted">{usage.name}</span>{/if}
          <span class="ml-auto inline-flex items-center gap-1.5 text-[11px] font-medium {usage.active ? 'text-success' : 'text-danger'}">
            <span class="w-1.5 h-1.5 rounded-full {usage.active ? 'bg-success' : 'bg-danger'}"></span>
            {usage.active ? 'Active' : 'Paused'}
          </span>
        </div>

        {#if usage.quotaExceeded}
          <div class="rounded-xl border border-danger/30 bg-danger/10 text-danger px-4 py-3 text-sm font-medium">
            Quota exhausted — please contact the administrator to top up your tokens.
          </div>
        {/if}

        <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div class="rounded-xl border border-border bg-surface p-4">
            <p class="text-[11px] uppercase tracking-wider text-text-subtle font-code">Quota</p>
            <p class="text-2xl font-bold mt-1">{usage.unlimited ? '∞' : fmtTokens(usage.tokenQuota ?? 0)}</p>
          </div>
          <div class="rounded-xl border border-border bg-surface p-4">
            <p class="text-[11px] uppercase tracking-wider text-text-subtle font-code">Used</p>
            <p class="text-2xl font-bold mt-1">{fmtTokens(usage.usedTokens ?? 0)}</p>
          </div>
          <div class="rounded-xl border border-border bg-surface p-4">
            <p class="text-[11px] uppercase tracking-wider text-text-subtle font-code">Remaining</p>
            <p class="text-2xl font-bold mt-1 {usage.unlimited ? '' : (percent >= 90 ? 'text-danger' : '')}">
              {usage.unlimited ? '∞' : fmtTokens(usage.remainingTokens ?? 0)}
            </p>
          </div>
        </div>

        {#if !usage.unlimited}
          <div>
            <div class="h-2 rounded-full bg-surface-2 overflow-hidden">
              <div class="h-full {barColor} transition-all" style="width: {percent}%"></div>
            </div>
            <p class="mt-1 text-[11px] text-text-subtle">{percent}% of your lifetime quota used</p>
          </div>
        {/if}
      </section>
    {/if}

    {#if usage && modelsList.length > 0}
      <section class="rounded-xl border border-border bg-surface p-4">
        <h2 class="text-sm font-semibold mb-3">Models available for this key ({modelsList.length})</h2>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-1.5">
          {#each modelsList as m (m.id)}
            <div class="flex items-center gap-2 text-xs font-code text-text-muted bg-surface-2/60 rounded px-2 py-1.5">
              <span class="w-1 h-1 rounded-full bg-success shrink-0"></span>
              <span class="truncate">{m.id}</span>
            </div>
          {/each}
        </div>
      </section>
    {/if}

    {#if usage && !usage.active}
      <p class="text-[11px] text-text-subtle">This key is paused. Usage history is shown but the key cannot serve requests.</p>
    {/if}
  </main>
</div>
