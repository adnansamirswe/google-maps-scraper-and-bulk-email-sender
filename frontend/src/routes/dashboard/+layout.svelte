<script lang="ts">
	import { onMount } from 'svelte';
	import { isAuthenticated, clearToken } from '$lib/api';
	import { page } from '$app/state';
	import AlertProvider from '$lib/AlertProvider.svelte';

	let { children } = $props();
	let sidebarCollapsed = $state(false);

	onMount(() => {
		if (!isAuthenticated()) {
			window.location.href = '/login';
			return;
		}
	});

	let currentPath = $derived(page.url.pathname);

	function logout() {
		clearToken();
		window.location.href = '/login';
	}

	const navItems = [
		{ path: '/dashboard', icon: 'add_circle', label: 'New Scrape' },
		{ path: '/dashboard/jobs', icon: 'list_alt', label: 'Job History' },
		{ path: '/dashboard/emailer', icon: 'mail', label: 'Bulk Emailer' },
	];
</script>

<svelte:head>
	<title>Dashboard — GMapScraper</title>
</svelte:head>

<AlertProvider />

<div class="dashboard-layout">
	<!-- Sidebar -->
	<aside class="sidebar" class:collapsed={sidebarCollapsed}>
		<div class="sidebar-header">
			<div class="sidebar-logo">
				{#if !sidebarCollapsed}
					<span class="logo-text">GMapScraper</span>
				{/if}
			</div>
			<button class="btn-icon" onclick={() => sidebarCollapsed = !sidebarCollapsed}>
				<span class="mi">{sidebarCollapsed ? 'chevron_right' : 'chevron_left'}</span>
			</button>
		</div>

		<nav class="sidebar-nav">
			{#each navItems as item}
				<a
					href={item.path}
					class="nav-item"
					class:active={currentPath === item.path}
				>
					<span class="mi">{item.icon}</span>
					{#if !sidebarCollapsed}
						<span class="nav-label">{item.label}</span>
					{/if}
				</a>
			{/each}
		</nav>

		<div class="sidebar-footer">
			<button class="nav-item logout-btn" onclick={logout}>
				<span class="mi">logout</span>
				{#if !sidebarCollapsed}
					<span class="nav-label">Logout</span>
				{/if}
			</button>
		</div>
	</aside>

	<!-- Main Content -->
	<main class="main-content">
		<div class="content-wrapper">
			{@render children()}
		</div>
	</main>
</div>

<style>
	.dashboard-layout {
		display: flex;
		min-height: 100vh;
	}

	/* Sidebar */
	.sidebar {
		width: 260px;
		background: var(--bg-secondary);
		border-right: 1px solid var(--border);
		display: flex;
		flex-direction: column;
		transition: width var(--transition-normal);
		flex-shrink: 0;
		position: sticky;
		top: 0;
		height: 100vh;
	}

	.sidebar.collapsed {
		width: 72px;
	}

	.sidebar-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px;
		border-bottom: 1px solid var(--border);
		height: 64px;
	}

	.sidebar-logo {
		display: flex;
		align-items: center;
		gap: 10px;
		overflow: hidden;
	}

	.logo-icon-sm {
		width: 36px;
		height: 36px;
		border-radius: 8px;
		background: linear-gradient(135deg, var(--accent-1), var(--accent-3));
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		flex-shrink: 0;
	}

	.logo-text {
		font-weight: 700;
		font-size: 1rem;
		white-space: nowrap;
		background: linear-gradient(135deg, var(--text-primary), var(--accent-2));
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}

	.sidebar-nav {
		flex: 1;
		padding: 8px;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.nav-item {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 10px 12px;
		border-radius: var(--radius-sm);
		color: var(--text-secondary);
		text-decoration: none;
		font-size: 0.875rem;
		font-weight: 500;
		transition: all var(--transition-fast);
		cursor: pointer;
		border: none;
		background: none;
		width: 100%;
		text-align: left;
		font-family: inherit;
	}

	.nav-item:hover {
		background: var(--bg-card);
		color: var(--text-primary);
	}

	.nav-item.active {
		background: rgba(108, 92, 231, 0.12);
		color: var(--accent-2);
	}

	.nav-item.active .mi {
		color: var(--accent-2);
	}

	.nav-label {
		white-space: nowrap;
		overflow: hidden;
	}

	.sidebar-footer {
		padding: 8px;
		border-top: 1px solid var(--border);
	}

	.logout-btn {
		color: var(--text-muted);
	}
	.logout-btn:hover {
		color: #e74c3c;
		background: rgba(231, 76, 60, 0.08);
	}

	/* Main Content */
	.main-content {
		flex: 1;
		min-width: 0;
		background: var(--bg-primary);
	}

	.content-wrapper {
		padding: 28px 36px;
	}

	/* Responsive */
	@media (max-width: 768px) {
		.sidebar {
			width: 72px;
		}
		.nav-label, .logo-text {
			display: none;
		}
		.content-wrapper {
			padding: 16px;
		}
	}
</style>
