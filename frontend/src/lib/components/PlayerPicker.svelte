<script lang="ts">
	import type { Player } from '$lib/tips.svelte';
	import { tipsStore } from '$lib/tips.svelte';
	import Flag from './Flag.svelte';
	import { X, Check } from '@lucide/svelte';

	let {
		players,
		value = $bindable(''),
		locked = false,
		placeholder = 'Search player…'
	}: {
		players: Player[];
		value: string;
		locked?: boolean;
		placeholder?: string;
	} = $props();

	let search = $state('');
	let open = $state(false);
	let inputEl: HTMLInputElement | undefined;

	$effect(() => {
		// Keep search in sync with external value changes.
		search = value;
	});

	let filtered = $derived.by(() => {
		const q = search.trim().toLowerCase();
		if (!q) return players.slice(0, 40);
		return players
			.filter((p) => {
				const team = tipsStore.teams[p.teamId];
				const tname = team?.name?.toLowerCase() ?? '';
				return p.name.toLowerCase().includes(q) || tname.includes(q);
			})
			.slice(0, 40);
	});

	let pickedPlayer = $derived(
		value && value === search ? (players.find((p) => p.name === value) ?? null) : null
	);
	let pickedTeam = $derived(pickedPlayer ? (tipsStore.teams[pickedPlayer.teamId] ?? null) : null);

	function pick(player: Player) {
		value = player.name;
		search = player.name;
		open = false;
	}

	function clear() {
		value = '';
		search = '';
		open = false;
		inputEl?.focus();
	}
</script>

<div class="pp">
	{#if pickedPlayer}
		<!-- Committed pick -->
		<div class="pp-pick">
			{#if pickedTeam}
				<Flag iso2={pickedTeam.iso2} code={pickedTeam.fifaCode} />
			{/if}
			<span class="pp-name">{value}</span>
			{#if pickedTeam}
				<span class="pp-team muted">{pickedTeam.name}</span>
			{/if}
			{#if !locked}
				<button class="pp-clear" onclick={clear} aria-label="Clear pick">
					<X size={14} />
				</button>
			{/if}
		</div>
	{/if}

	{#if !locked}
		<div class="pp-search">
			<input
				bind:this={inputEl}
				class="pp-input"
				type="text"
				{placeholder}
				bind:value={search}
				onfocus={() => (open = true)}
				onblur={() => setTimeout(() => (open = false), 160)}
				oninput={() => {
					if (search !== value) value = '';
					open = true;
				}}
				disabled={locked}
			/>
			{#if open && filtered.length > 0}
				<ul class="pp-dropdown">
					{#each filtered as p (p.id)}
						{@const team = tipsStore.teams[p.teamId]}
						<li>
							<button
								onmousedown={(e) => e.preventDefault()}
								onclick={() => pick(p)}
								class:sel={value === p.name}
							>
								<span class="pp-row-name">
									{#if team}
										<Flag iso2={team.iso2} code={team.fifaCode} />
									{/if}
									<span>{p.name}</span>
								</span>
								<span class="pp-row-meta">
									{#if team}<span class="pp-row-team muted">{team.name}</span>{/if}
									<span class="pp-pos muted">{p.position}</span>
								</span>
								{#if value === p.name}
									<Check size={13} class="pp-check" />
								{/if}
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	{/if}
</div>

<style>
	.pp {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.pp-pick {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.5rem 0.75rem;
		border: 1px solid color-mix(in srgb, var(--success) 40%, var(--border));
		border-radius: var(--radius-sm);
		background: color-mix(in srgb, var(--success) 8%, var(--surface-2));
		font-size: 0.88rem;
	}
	.pp-name {
		font-weight: 700;
	}
	.pp-team {
		font-size: 0.78rem;
	}
	.pp-clear {
		margin-left: auto;
		background: none;
		border: none;
		color: var(--muted);
		cursor: pointer;
		padding: 0.1rem;
		display: flex;
		align-items: center;
	}
	.pp-clear:hover {
		color: var(--text);
	}
	.pp-search {
		position: relative;
	}
	.pp-input {
		width: 100%;
		padding: 0.65rem 0.85rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface-2);
		color: var(--text);
		font: inherit;
		font-size: 0.88rem;
		box-sizing: border-box;
	}
	.pp-input:focus {
		outline: 2px solid color-mix(in srgb, var(--accent) 50%, transparent);
		outline-offset: -1px;
	}
	.pp-dropdown {
		position: absolute;
		left: 0;
		right: 0;
		top: calc(100% + 2px);
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		box-shadow: var(--shadow-pop);
		list-style: none;
		margin: 0;
		padding: 0.15rem 0;
		max-height: 380px;
		overflow-y: auto;
		z-index: 50;
	}
	.pp-dropdown li button {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		width: 100%;
		padding: 0.38rem 0.7rem;
		background: none;
		border: none;
		color: var(--text);
		font: inherit;
		font-size: 0.85rem;
		cursor: pointer;
		text-align: left;
	}
	.pp-dropdown li button:hover,
	.pp-dropdown li button.sel {
		background: color-mix(in srgb, var(--accent) 10%, transparent);
	}
	.pp-row-name {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		font-weight: 600;
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.pp-row-meta {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		flex-shrink: 0;
	}
	.pp-row-team {
		font-size: 0.75rem;
		white-space: nowrap;
	}
	.pp-pos {
		font-size: 0.7rem;
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		flex-shrink: 0;
	}
	:global(.pp-check) {
		color: var(--success);
		flex-shrink: 0;
	}
</style>
