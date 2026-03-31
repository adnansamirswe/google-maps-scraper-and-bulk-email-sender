<script lang="ts">
	import { alerts } from './alerts.svelte';
	import { fade, fly } from 'svelte/transition';
</script>

<!-- Toasts Container -->
<div class="toast-container">
	{#each alerts.toasts as toast (toast.id)}
		<div 
			class="toast toast-{toast.type}"
			in:fly={{ y: 20, duration: 400 }}
			out:fade={{ duration: 200 }}>
			<span class="mi mi-sm">
				{#if toast.type === 'success'}check_circle
				{:else if toast.type === 'error'}report
				{:else if toast.type === 'warning'}warning
				{:else}info{/if}
			</span>
			<span class="toast-message">{toast.message}</span>
		</div>
	{/each}
</div>

<!-- Confirmation Modal -->
{#if alerts.confirmation}
	<div class="modal-backdrop" transition:fade={{ duration: 200 }}>
		<div class="modal animate-pop-in">
			<div class="modal-header">
				<h3 class="flex items-center gap-2">
					<span class="mi text-accent">help_outline</span>
					{alerts.confirmation.title}
				</h3>
			</div>
			<div class="modal-body">
				<p class="text-secondary">{alerts.confirmation.message}</p>
			</div>
			<div class="modal-footer">
				<button class="btn btn-secondary" onclick={() => alerts.confirmation?.resolve(false)}>
					Cancel
				</button>
				<button class="btn btn-primary" onclick={() => alerts.confirmation?.resolve(true)}>
					Confirm
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.toast-container {
		position: fixed;
		bottom: 24px;
		right: 24px;
		display: flex;
		flex-direction: column;
		gap: 12px;
		z-index: 9999;
		pointer-events: none;
	}

	.toast {
		pointer-events: auto;
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 16px;
		border-radius: 12px;
		background: rgba(30, 30, 45, 0.85);
		backdrop-filter: blur(12px);
		border: 1px solid rgba(255, 255, 255, 0.1);
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
		min-width: 280px;
		max-width: 400px;
		color: white;
	}

	.toast-success { border-left: 4px solid #10b981; }
	.toast-error { border-left: 4px solid #ef4444; }
	.toast-warning { border-left: 4px solid #f59e0b; }
	.toast-info { border-left: 4px solid #3b82f6; }

	.toast-success .mi { color: #10b981; }
	.toast-error .mi { color: #ef4444; }
	.toast-warning .mi { color: #f59e0b; }
	.toast-info .mi { color: #3b82f6; }

	.toast-message {
		font-size: 0.9rem;
		font-weight: 500;
	}

	.animate-pop-in {
		animation: popIn 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
	}

	@keyframes popIn {
		from { transform: scale(0.85); opacity: 0; }
		to { transform: scale(1); opacity: 1; }
	}
</style>
