<script lang="ts">
	import { onMount } from 'svelte';
	import { isAuthenticated } from '$lib/api';

	onMount(() => {
		if (isAuthenticated()) {
			window.location.href = '/dashboard';
		} else {
			window.location.href = '/login';
		}
	});
</script>

<div class="loading-screen">
	<div class="spinner"></div>
</div>

<style>
	.loading-screen {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 100vh;
		background: var(--bg-primary);
	}
	.spinner {
		width: 40px;
		height: 40px;
		border: 3px solid var(--border);
		border-top-color: var(--accent-1);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}
</style>
