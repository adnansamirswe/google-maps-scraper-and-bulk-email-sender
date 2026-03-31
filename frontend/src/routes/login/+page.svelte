<script lang="ts">
	import { login, isAuthenticated } from '$lib/api';
	import { onMount } from 'svelte';

	let password = $state('');
	let error = $state('');
	let loading = $state(false);
	let shaking = $state(false);

	onMount(() => {
		if (isAuthenticated()) {
			window.location.href = '/dashboard';
		}
	});

	async function handleLogin(e: Event) {
		e.preventDefault();
		if (!password.trim()) {
			error = 'Please enter a password';
			shaking = true;
			setTimeout(() => shaking = false, 500);
			return;
		}

		loading = true;
		error = '';

		try {
			await login(password);
			window.location.href = '/dashboard';
		} catch (err: any) {
			error = err.message || 'Wrong password';
			shaking = true;
			setTimeout(() => shaking = false, 500);
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Login — GMapScraper</title>
</svelte:head>

<div class="login-page">
	<!-- Animated background orbs -->
	<div class="bg-orb orb-1"></div>
	<div class="bg-orb orb-2"></div>
	<div class="bg-orb orb-3"></div>

	<div class="login-container animate-fade-in-up">
		<div class="login-card card-glass" class:animate-shake={shaking}>
			<!-- Logo Section -->
			<div class="logo-section">
				<h1>GMapScraper</h1>
				<p class="text-secondary">Elevate your lead generation strategy</p>
			</div>

			<!-- Login Form -->
			<form onsubmit={handleLogin} class="login-form">
				<div class="input-group">
					<label for="password-input">Password</label>
					<div class="password-input-wrapper">
						<span class="mi mi-sm input-icon">lock</span>
						<input
							id="password-input"
							type="password"
							class="input password-field"
							placeholder="Enter your password"
							bind:value={password}
							disabled={loading}
							autocomplete="current-password"
						/>
					</div>
				</div>

				{#if error}
					<div class="error-msg animate-fade-in">
						<span class="mi mi-sm">error</span>
						{error}
					</div>
				{/if}

				<button type="submit" class="btn btn-primary w-full login-btn" disabled={loading}>
					{#if loading}
						<div class="btn-spinner"></div>
						Authenticating...
					{:else}
						<span class="mi mi-sm">login</span>
						Enter Dashboard
					{/if}
				</button>
			</form>

			<!-- Footer -->
			<div class="login-footer">
				<span class="mi mi-sm">schedule</span>
				<span>Session expires after 7 days</span>
			</div>
		</div>
	</div>
</div>

<style>
	.login-page {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
		position: relative;
		overflow: hidden;
		background: radial-gradient(ellipse at 20% 50%, rgba(108, 92, 231, 0.08) 0%, transparent 50%),
					radial-gradient(ellipse at 80% 20%, rgba(0, 206, 201, 0.06) 0%, transparent 50%),
					var(--bg-primary);
	}

	/* Floating orbs */
	.bg-orb {
		position: absolute;
		border-radius: 50%;
		filter: blur(80px);
		opacity: 0.4;
		animation: float 20s ease-in-out infinite;
	}
	.orb-1 {
		width: 500px;
		height: 500px;
		background: radial-gradient(circle, var(--accent-1) 0%, transparent 70%);
		top: -250px;
		right: -100px;
		animation-delay: 0s;
		filter: blur(100px);
	}
	.orb-2 {
		width: 400px;
		height: 400px;
		background: radial-gradient(circle, var(--accent-3) 0%, transparent 70%);
		bottom: -200px;
		left: -150px;
		animation-delay: -7s;
		filter: blur(80px);
	}
	.orb-3 {
		width: 300px;
		height: 300px;
		background: radial-gradient(circle, var(--accent-2) 0%, transparent 70%);
		top: 40%;
		right: 20%;
		animation-delay: -14s;
		filter: blur(60px);
		opacity: 0.2;
	}

	@keyframes float {
		0%, 100% { transform: translate(0, 0) scale(1); }
		33% { transform: translate(30px, -30px) scale(1.05); }
		66% { transform: translate(-20px, 20px) scale(0.95); }
	}

	.login-container {
		position: relative;
		z-index: 1;
		width: 100%;
		max-width: 440px;
		padding: 20px;
	}

	.login-card {
		padding: 50px 40px;
		background: rgba(26, 26, 38, 0.45);
		backdrop-filter: blur(30px);
		-webkit-backdrop-filter: blur(30px);
		border: 1px solid rgba(255, 255, 255, 0.08);
		box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5),
					inset 0 1px 1px rgba(255, 255, 255, 0.1);
		transition: all 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
	}

	.login-card:hover {
		border-color: rgba(108, 92, 231, 0.3);
		transform: translateY(-5px);
		box-shadow: 0 35px 60px -15px rgba(0, 0, 0, 0.6),
					0 0 20px rgba(108, 92, 231, 0.15);
	}

	.logo-section h1 {
		font-size: 2.8rem;
		font-weight: 900;
		letter-spacing: -1.5px;
		background: linear-gradient(to right, #fff 20%, var(--accent-1), var(--accent-2));
		-webkit-background-clip: text;
		background-clip: text;
		-webkit-text-fill-color: transparent;
		margin-bottom: 12px;
		filter: drop-shadow(0 4px 8px rgba(0,0,0,0.3));
	}

	.logo-section p {
		font-size: 1rem;
		font-weight: 500;
		opacity: 0.7;
	}

	.login-form {
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.password-input-wrapper {
		position: relative;
		display: flex;
		align-items: center;
	}

	.input-icon {
		position: absolute;
		left: 12px;
		color: var(--text-muted);
		pointer-events: none;
	}

	.password-field {
		padding-left: 44px;
		width: 100%;
		height: 54px;
		font-size: 1.1rem;
		background: rgba(0, 0, 0, 0.3);
		border-radius: 12px;
		transition: all 0.3s ease;
	}

	.password-field:focus {
		background: rgba(0, 0, 0, 0.4);
		transform: scale(1.02);
		box-shadow: 0 0 0 4px rgba(108, 92, 231, 0.15);
	}

	.error-msg {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 14px;
		border-radius: var(--radius-sm);
		background: rgba(231, 76, 60, 0.1);
		border: 1px solid rgba(231, 76, 60, 0.2);
		color: #e74c3c;
		font-size: 0.85rem;
	}

	.login-btn {
		height: 48px;
		font-size: 0.95rem;
		justify-content: center;
	}

	.btn-spinner {
		width: 18px;
		height: 18px;
		border: 2px solid rgba(255, 255, 255, 0.3);
		border-top-color: white;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	.login-footer {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		margin-top: 32px;
		padding-top: 24px;
		border-top: 1px solid rgba(255, 255, 255, 0.05);
		color: var(--text-muted);
		font-size: 0.85rem;
		letter-spacing: 0.5px;
		text-transform: uppercase;
		opacity: 0.6;
	}
</style>
