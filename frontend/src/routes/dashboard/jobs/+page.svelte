<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { listJobs, deleteJob, getPlaces } from '$lib/api';
	import { exportToCSV } from '$lib/utils';
	import { alerts } from '$lib/alerts.svelte';

	let jobs = $state<any[]>([]);
	let loading = $state(true);
	let expandedJob = $state<string | null>(null);
	let jobPlaces = $state<Record<string, any[]>>({});
	let loadingPlaces = $state<Record<string, boolean>>({});
	let pollInterval: ReturnType<typeof setInterval> | null = null;

	let copyStatus = $state<Record<string, boolean>>({});

	async function copyColumn(id: string, places: any[], column: string) {
		const data = places.map(p => p[column]).filter(Boolean).join('\n');
		if (!data) {
			alerts.warning(`No valid ${column}s available to copy!`);
			return;
		}
		try {
			if (navigator.clipboard && navigator.clipboard.writeText) {
				await navigator.clipboard.writeText(data);
			} else {
				const textArea = document.createElement("textarea");
				textArea.value = data;
				document.body.appendChild(textArea);
				textArea.select();
				document.execCommand("copy");
				textArea.remove();
			}
			copyStatus[id] = true;
			alerts.success(`Copied ${column}s to clipboard!`);
			setTimeout(() => { copyStatus[id] = false; }, 2000);
		} catch (err) {
			console.error('Failed to copy', err);
			alerts.error('Failed to copy to clipboard.');
		}
	}

	onMount(async () => {
		await refreshJobs();
		startPolling();
	});

	onDestroy(() => {
		stopPolling();
	});

	function startPolling() {
		pollInterval = setInterval(async () => {
			// Only poll if at least one job is running
			const hasRunning = jobs.some(j => j.status === 'running');
			if (hasRunning) {
				await refreshJobs(true);
			}
		}, 5000);
	}

	function stopPolling() {
		if (pollInterval) {
			clearInterval(pollInterval);
			pollInterval = null;
		}
	}

	async function refreshJobs(silent = false) {
		if (!silent) loading = true;
		try {
			jobs = await listJobs();
		} catch {
			jobs = [];
		} finally {
			loading = false;
		}
	}

	async function handleDelete(e: MouseEvent, id: string) {
		e.stopPropagation();
		const confirmed = await alerts.confirm('Delete Job', 'Are you sure you want to permanently delete this job and all its results?');
		if (!confirmed) return;
	
		try {
			await deleteJob(id);
			jobs = jobs.filter(j => j.id !== id);
			alerts.success('Job deleted successfully.');
		} catch {
			alerts.error('Failed to delete job');
		}
	}

	async function toggleExpand(id: string) {
		if (expandedJob === id) {
			expandedJob = null;
			return;
		}

		expandedJob = id;

		if (!jobPlaces[id]) {
			loadingPlaces = { ...loadingPlaces, [id]: true };
			try {
				const places = await getPlaces(id);
				jobPlaces = { ...jobPlaces, [id]: places };
			} catch {
				jobPlaces = { ...jobPlaces, [id]: [] };
			}
			loadingPlaces = { ...loadingPlaces, [id]: false };
		}
	}

	function formatDate(dateStr: string) {
		try {
			return new Date(dateStr).toLocaleString('en-US', {
				month: 'short',
				day: 'numeric',
				hour: '2-digit',
				minute: '2-digit',
			});
		} catch {
			return dateStr;
		}
	}

	function statusBadgeClass(status: string) {
		return `badge badge-${status}`;
	}
</script>

<div class="page-header">
	<div class="flex justify-between items-center">
		<div>
			<h1>
				<span class="mi mi-lg">list_alt</span>
				Job History
			</h1>
			<p class="text-secondary">View and manage your scraping jobs</p>
		</div>
		<button class="btn btn-secondary" onclick={refreshJobs} disabled={loading}>
			<span class="mi mi-sm">refresh</span>
			Refresh
		</button>
	</div>
</div>

{#if loading}
	<div class="card">
		<div class="progress-bar progress-bar-indeterminate">
			<div class="progress-bar-fill"></div>
		</div>
		<p class="text-center text-secondary mt-4">Loading jobs...</p>
	</div>
{:else if jobs.length === 0}
	<div class="card">
		<div class="empty-state">
			<span class="mi">history</span>
			<h3>No scraping history</h3>
			<p>Your previous extraction jobs will appear here once they finish processing.</p>
			<a href="/dashboard" class="btn btn-primary">
				<span class="mi mi-sm">add</span>
				Start Extracting
			</a>
		</div>
	</div>
{:else}
	<div class="jobs-list">
		{#each jobs as job (job.id)}
			<div class="job-card card animate-fade-in" class:expanded={expandedJob === job.id}>
				<!-- Job Header -->
				<div 
					class="job-header" 
					role="button" 
					tabindex="0" 
					onclick={() => toggleExpand(job.id)}
					onkeydown={(e) => e.key === 'Enter' && toggleExpand(job.id)}
				>
					<div class="job-info">
						<div class="flex items-center gap-3">
							<h3>{job.name}</h3>
							<span class={statusBadgeClass(job.status)}>{job.status}</span>
						</div>
						<div class="job-meta flex items-center gap-4">
							<span class="flex items-center gap-1 text-sm text-secondary">
								<span class="mi mi-sm">schedule</span>
								{formatDate(job.created_at)}
							</span>
							<span class="flex items-center gap-1 text-sm text-secondary">
								<span class="mi mi-sm">place</span>
								{job.places_found} places
							</span>
							<span class="flex items-center gap-1 text-sm text-secondary">
								<span class="mi mi-sm">search</span>
								{job.keywords?.length || 0} keywords
							</span>
						</div>
					</div>
					<div class="job-actions flex items-center gap-2">
						<button
							class="btn-icon"
							onclick={(e) => handleDelete(e, job.id)}
							title="Delete job"
						>
							<span class="mi" style="color: #e74c3c;">delete</span>
						</button>
						<span class="mi expand-icon">{expandedJob === job.id ? 'expand_less' : 'expand_more'}</span>
					</div>
				</div>

				<!-- Expanded Content -->
				{#if expandedJob === job.id}
					<div class="job-details animate-fade-in">
						<!-- Fields used -->
						<div class="detail-section">
							<h4 class="text-secondary text-sm mb-2">Selected Fields</h4>
							<div class="chips-wrap">
								{#each Object.entries(job.fields || {}) as [key, value]}
									{#if value}
										<span class="chip chip-accent">{key.replace('_', ' ')}</span>
									{/if}
								{/each}
							</div>
						</div>

						<!-- Keywords -->
						<div class="detail-section">
							<h4 class="text-secondary text-sm mb-2">Keywords</h4>
							<div class="chips-wrap">
								{#each (job.keywords || []) as kw}
									<span class="chip">{kw}</span>
								{/each}
							</div>
						</div>

						<!-- Places Table -->
						<div class="detail-section">
							<div class="flex items-center justify-between flex-wrap gap-2 mb-4">
								<h4 class="text-secondary text-sm m-0">Results ({job.places_found} places)</h4>
								{#if (jobPlaces[job.id]?.length ?? 0) > 0}
									<div class="flex items-center gap-2">
										<button style="font-size: 0.9rem;" class="btn btn-secondary btn-sm" onclick={(e) => { e.stopPropagation(); copyColumn(`job_${job.id}_email`, jobPlaces[job.id], 'email'); }}>
											<span class="mi mi-sm">{copyStatus[`job_${job.id}_email`] ? 'check' : 'content_copy'}</span>
											{copyStatus[`job_${job.id}_email`] ? 'Copied' : 'Emails'}
										</button>
										<button style="font-size: 0.9rem;" class="btn btn-secondary btn-sm" onclick={(e) => { e.stopPropagation(); copyColumn(`job_${job.id}_phone`, jobPlaces[job.id], 'phone'); }}>
											<span class="mi mi-sm">{copyStatus[`job_${job.id}_phone`] ? 'check' : 'content_copy'}</span>
											{copyStatus[`job_${job.id}_phone`] ? 'Copied' : 'Phones'}
										</button>
										<button style="font-size: 0.9rem;" class="btn btn-secondary btn-sm" onclick={(e) => { e.stopPropagation(); exportToCSV(`job_${job.id}_results.csv`, jobPlaces[job.id]); }}>
											<span class="mi mi-sm">download</span>
											Export CSV
										</button>
									</div>
								{/if}
							</div>

							{#if loadingPlaces[job.id]}
								<div class="progress-bar progress-bar-indeterminate" style="margin: 16px 0;">
									<div class="progress-bar-fill"></div>
								</div>
							{:else if (jobPlaces[job.id]?.length ?? 0) > 0}
								<div class="results-table-wrapper">
									<table class="data-table">
										<thead>
											<tr>
												<th>#</th>
												<th>Name</th>
												{#if job.fields?.category}<th>Category</th>{/if}
												{#if job.fields?.phone}<th>Phone</th>{/if}
												{#if job.fields?.email}<th>Email</th>{/if}
												{#if job.fields?.website}<th>Website</th>{/if}
												{#if job.fields?.review_rating}<th>Rating</th>{/if}
												{#if job.fields?.reviews_per_rating}<th>Breakdown</th>{/if}
												{#if job.fields?.review_count}<th>Reviews</th>{/if}
												{#if job.fields?.address}<th>Address</th>{/if}
											</tr>
										</thead>
										<tbody>
											{#each jobPlaces[job.id] as place, i}
												<tr>
													<td class="text-muted">{i + 1}</td>
													<td><strong>{place.title}</strong></td>
													{#if job.fields?.category}
														<td><span class="chip">{place.category || '—'}</span></td>
													{/if}
													{#if job.fields?.phone}
														<td class="font-mono text-sm">{place.phone || '—'}</td>
													{/if}
													{#if job.fields?.email}
														<td class="text-accent text-sm">
															{#if place.email}
																<div class="truncate-clickable" onclick={(e) => e.currentTarget.classList.toggle('expanded')} title={place.email}>{place.email}</div>
															{:else}
																—
															{/if}
														</td>
													{/if}
													{#if job.fields?.website}
														<td>
															{#if place.website}
																<div class="truncate-clickable" onclick={(e) => e.currentTarget.classList.toggle('expanded')} title={place.website}>
																	<a href={place.website} target="_blank" class="link">{place.website.replace('https://', '').replace('http://', '')}</a>
																</div>
															{:else}
																—
															{/if}
														</td>
													{/if}
													{#if job.fields?.review_rating}
														<td>
															{#if place.review_rating}
																<span class="rating">
																	<span class="mi mi-sm" style="color: #f39c12;">star</span>
																	{place.review_rating.toFixed(1)}
																</span>
															{:else}
																—
															{/if}
														</td>
													{/if}
													{#if job.fields?.reviews_per_rating}
													<td>
														{#if place.reviews_per_rating && Object.keys(place.reviews_per_rating).length > 0}
															<div class="flex items-center gap-2">
																{#each [5,4,3,2,1] as stars}
																	{#if place.reviews_per_rating[stars]}
																		<span class="text-xs text-secondary bg-gray-100 px-1 rounded dark:bg-gray-800" title="{stars} Stars">{stars}★:{place.reviews_per_rating[stars]}</span>
																	{/if}
																{/each}
															</div>
														{:else}
															—
														{/if}
													</td>
													{/if}
													{#if job.fields?.review_count}
													<td>{place.review_count || 0}</td>
													{/if}
													{#if job.fields?.address}
													<td>
														{#if place.address}
															<div class="truncate-clickable text-sm" style="max-width: 200px;" title={place.address} onclick={(e) => e.currentTarget.classList.toggle('expanded')}>{place.address}</div>
														{:else}
															—
														{/if}
													</td>
													{/if}
												</tr>
											{/each}
										</tbody>
									</table>
								</div>
							{:else}
								<p class="text-muted text-sm">No results yet. The job may still be running.</p>
							{/if}
						</div>
					</div>
				{/if}
			</div>
		{/each}
	</div>
{/if}

<style>
	.page-header {
		margin-bottom: 28px;
	}

	.page-header h1 {
		display: flex;
		align-items: center;
		gap: 10px;
		font-size: 1.6rem;
		font-weight: 700;
		margin-bottom: 4px;
	}

	.page-header h1 .mi {
		color: var(--accent-2);
	}

	.text-center {
		text-align: center;
	}

	.jobs-list {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.job-card {
		padding: 0;
		overflow: hidden;
	}

	.job-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		width: 100%;
		padding: 16px 20px;
		background: none;
		border: none;
		cursor: pointer;
		font-family: inherit;
		color: inherit;
		text-align: left;
	}

	.job-header:hover {
		background: var(--bg-card-hover);
	}

	.job-info {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.job-info h3 {
		font-size: 1rem;
		font-weight: 600;
	}

	.expand-icon {
		color: var(--text-muted);
		transition: transform var(--transition-fast);
	}

	.job-details {
		padding: 0 20px 20px;
		border-top: 1px solid var(--border);
	}

	.detail-section {
		margin-top: 16px;
	}

	.chips-wrap {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
	}

	.results-table-wrapper {
		overflow-x: auto;
		border-radius: var(--radius-sm);
		border: 1px solid var(--border);
	}

	.rating {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-weight: 600;
		font-size: 0.9rem;
	}

	.link {
		color: var(--accent-2);
		text-decoration: none;
	}
	.link:hover {
		text-decoration: underline;
	}
</style>
