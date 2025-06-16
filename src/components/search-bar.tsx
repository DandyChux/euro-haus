import { useState, useEffect, useRef, useCallback } from "react"
import { Input } from "~/components/ui/input"
import { Button } from "~/components/ui/button"
import { Card } from "~/components/ui/card"
import { Badge } from "~/components/ui/badge"
import { Skeleton } from "~/components/ui/skeleton"
import { Search, X, Package, Calendar } from "lucide-react"
import { useNavigate } from "@tanstack/react-router"
import { apiClient } from "~/lib/api"

interface StripeProduct {
	id: string
	name: string
	description: string | null
	images: string[]
	metadata: Record<string, string>
	active: boolean
	default_price: {
		id: string
		unit_amount: number
		currency: string
	} | null
}

interface SearchBarProps {
	onSearch: (query: string) => void
	enableProductSearch?: boolean // Optional prop to enable Stripe product search
	placeholder?: string
}

export function SearchBar({ onSearch, enableProductSearch = false, placeholder = "Search..." }: SearchBarProps) {
	const [searchQuery, setSearchQuery] = useState<string>("")
	const [isExpanded, setIsExpanded] = useState<boolean>(false)
	const [isLoading, setIsLoading] = useState<boolean>(false)
	const [results, setResults] = useState<StripeProduct[]>([])
	const [showResults, setShowResults] = useState<boolean>(false)
	const searchRef = useRef<HTMLDivElement>(null)
	const debounceTimerRef = useRef<NodeJS.Timeout | undefined>(undefined)
	const navigate = useNavigate()

	// Close dropdown when clicking outside
	useEffect(() => {
		if (!enableProductSearch) return

		const handleClickOutside = (event: MouseEvent) => {
			if (searchRef.current && !searchRef.current.contains(event.target as Node)) {
				setShowResults(false)
			}
		}

		document.addEventListener("mousedown", handleClickOutside)
		return () => document.removeEventListener("mousedown", handleClickOutside)
	}, [enableProductSearch])

	// Debounced search function
	const searchProducts = useCallback(async (query: string) => {
		if (!enableProductSearch || !query.trim()) {
			setResults([])
			setShowResults(false)
			return
		}

		setIsLoading(true)
		try {
			const response = await apiClient.get('/products')
			const allProducts: StripeProduct[] = response.data.products || []

			// Filter products based on search query
			const filtered = allProducts.filter(product =>
				product.name.toLowerCase().includes(query.toLowerCase()) ||
				(product.description && product.description.toLowerCase().includes(query.toLowerCase()))
			).slice(0, 8) // Limit to 8 results

			setResults(filtered)
			setShowResults(filtered.length > 0)
		} catch (error) {
			console.error('Error searching products:', error)
			setResults([])
		} finally {
			setIsLoading(false)
		}
	}, [enableProductSearch])

	const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		const newQuery = e.target.value
		setSearchQuery(newQuery)
		onSearch(newQuery) // Always call the onSearch callback

		// Only search products if enabled
		if (enableProductSearch) {
			// Clear previous timer
			if (debounceTimerRef.current) {
				clearTimeout(debounceTimerRef.current)
			}

			// Set new timer for debounced search
			if (newQuery.trim()) {
				setIsLoading(true)
				debounceTimerRef.current = setTimeout(() => {
					searchProducts(newQuery)
				}, 300)
			} else {
				setResults([])
				setShowResults(false)
				setIsLoading(false)
			}
		}
	}

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault()

		// If product search is enabled and we have results, navigate to first result
		if (enableProductSearch && searchQuery.trim() && results.length > 0) {
			handleProductClick(results[0])
		} else {
			// Otherwise just call onSearch
			onSearch(searchQuery)
		}
	}

	const handleProductClick = (product: StripeProduct) => {
		const isEvent = product.metadata.type === 'event'
		if (isEvent && product.metadata.slug) {
			navigate({ to: `/events/${product.metadata.slug}` })
		} else {
			navigate({ to: `/catalog/${product.id}` })
		}

		// Reset search
		setSearchQuery("")
		setShowResults(false)
		setIsExpanded(false)
	}

	const toggleSearch = () => {
		setIsExpanded(!isExpanded)
		// Focus the input when expanded
		if (!isExpanded) {
			setTimeout(() => document.getElementById("search-input")?.focus(), 100)
		} else {
			// Clear search when closing
			setSearchQuery("")
			setResults([])
			setShowResults(false)
		}
	}

	const formatPrice = (cents: number) => {
		return new Intl.NumberFormat('en-US', {
			style: 'currency',
			currency: 'USD',
		}).format(cents / 100)
	}

	return (
		<div ref={searchRef} className="relative">
			<form onSubmit={handleSubmit} className="flex items-center">
				<div className="relative flex items-center">
					{/* Search button first, then input - this helps with the left expansion */}
					<Button
						type="button"
						variant="ghost"
						onClick={toggleSearch}
						className="hover:cursor-pointer"
					>
						{isExpanded && enableProductSearch ? (
							<X className='h-4 w-4' />
						) : (
							<Search className='h-4 w-4' />
						)}
					</Button>
					<div className={`absolute right-full overflow-hidden transition-all duration-300 ${isExpanded ? (enableProductSearch ? "w-64 sm:w-72 md:w-80 pr-2" : "w-28 sm:w-40 md:w-56 pr-2") : "w-0"
						}`}>
						<Input
							id="search-input"
							type="text"
							placeholder={placeholder}
							value={searchQuery}
							onChange={handleInputChange}
							className='w-full'
						/>
					</div>
				</div>
			</form>

			{/* Search Results Dropdown - Only shown if enableProductSearch is true */}
			{enableProductSearch && isExpanded && showResults && (
				<Card className="absolute right-0 top-full mt-2 w-80 sm:w-96 max-h-96 overflow-y-auto shadow-lg z-50">
					{isLoading ? (
						<div className="p-4 space-y-3">
							{[...Array(3)].map((_, i) => (
								<div key={i} className="flex gap-3">
									<Skeleton className="h-16 w-16 rounded" />
									<div className="flex-1 space-y-2">
										<Skeleton className="h-4 w-3/4" />
										<Skeleton className="h-3 w-full" />
										<Skeleton className="h-3 w-1/4" />
									</div>
								</div>
							))}
						</div>
					) : (
						<div className="p-2">
							{results.map((product) => {
								const isEvent = product.metadata.type === 'event'
								return (
									<button
										key={product.id}
										onClick={() => handleProductClick(product)}
										className="w-full text-left p-2 hover:bg-accent rounded-md transition-colors"
									>
										<div className="flex gap-3">
											{product.images[0] ? (
												<img
													src={product.images[0]}
													alt={product.name}
													className="h-16 w-16 object-cover rounded"
												/>
											) : (
												<div className="h-16 w-16 bg-muted rounded flex items-center justify-center">
													{isEvent ? (
														<Calendar className="h-6 w-6 text-muted-foreground" />
													) : (
														<Package className="h-6 w-6 text-muted-foreground" />
													)}
												</div>
											)}
											<div className="flex-1 min-w-0">
												<h4 className="font-medium text-sm truncate">{product.name}</h4>
												{product.description && (
													<p className="text-xs text-muted-foreground line-clamp-2">
														{product.description}
													</p>
												)}
												<div className="flex items-center gap-2 mt-1">
													{product.default_price && (
														<span className="text-sm font-semibold">
															{formatPrice(product.default_price.unit_amount)}
														</span>
													)}
													<Badge variant={isEvent ? 'default' : 'secondary'} className="text-xs">
														{isEvent ? 'Event' : 'Product'}
													</Badge>
													{product.metadata.featured === 'true' && (
														<Badge variant="outline" className="text-xs">Featured</Badge>
													)}
												</div>
											</div>
										</div>
									</button>
								)
							})}
						</div>
					)}
				</Card>
			)}
		</div>
	)
}
