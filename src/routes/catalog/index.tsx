import React, { useEffect, useState } from 'react';
import { createFileRoute } from '@tanstack/react-router'
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbSeparator } from '~/components/ui/breadcrumb'
import { Button } from '~/components/ui/button';
import { Badge } from '~/components/ui/badge';
import { Filter, X } from 'lucide-react';
import { Separator } from '~/components/ui/separator';
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '~/components/ui/accordion';
import { Checkbox } from '~/components/ui/checkbox';
import { Label } from '~/components/ui/label';
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from '~/components/ui/sheet';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '~/components/ui/select';
import { ProductCard } from '~/components/product-card';
import { Loader2 } from 'lucide-react';
import { Product, stripeService } from '~/lib/services/stripe-service';

export const Route = createFileRoute('/catalog/')({
	component: RouteComponent,
	loader: async () => {
		const products = await stripeService.getAllProducts()

		if (!products) {
			throw new Error('Failed to load products. Please try again later.')
		}

		return products
	},
	pendingComponent: () => (
		<div className="min-h-screen flex items-center justify-center">
			<Loader2 className="h-8 w-8 animate-spin" />
		</div>
	),
	errorComponent: ({ error, reset }) => (
		<div className="min-h-screen flex items-center justify-center">
			<div className="text-center">
				<p className="text-destructive mb-4">{error.message}</p>
				<Button onClick={() => reset()}>Retry</Button>
			</div>
		</div>
	)
})

function RouteComponent() {
	const products = Route.useLoaderData()
	const [filteredProducts, setFilteredProducts] = useState<Product[]>([])
	const [activeCategory, setActiveCategory] = useState<string>("all")
	const [activeFilters, setActiveFilters] = useState<{
		categories: string[]
		subcategories: string[]
		tags: string[]
		price: { min: number; max: number } | null
		availability: string | null
	}>({
		categories: [],
		subcategories: [],
		tags: [],
		price: null,
		availability: null,
	})
	const [sortOption, setSortOption] = useState<string>("featured")
	const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false)

	// Get unique categories, subcategories, and tags
	const categories = [...new Set(products.map(p => p.category).filter((c): c is string => c !== undefined))].sort()
	const subcategories = [...new Set(products.map(p => p.subcategory).filter((s): s is string => s !== undefined))].sort()
	const allTags = [...new Set(products.flatMap(p => p.tags || []))].sort()

	// Price range
	const priceRange = products.length > 0 ? {
		min: Math.min(...products.map(p => p.price)),
		max: Math.max(...products.map(p => p.price))
	} : { min: 0, max: 1000 }

	// Apply filters and sorting
	useEffect(() => {
		let filtered = [...products]

		// Category filter from tabs
		if (activeCategory !== "all") {
			filtered = filtered.filter(p => p.category?.toLowerCase() === activeCategory.toLowerCase())
		}

		// Advanced filters
		if (activeFilters.categories.length > 0) {
			filtered = filtered.filter(p => activeFilters.categories.includes(p.category || ''))
		}

		if (activeFilters.subcategories.length > 0) {
			filtered = filtered.filter(p => activeFilters.subcategories.includes(p.subcategory || ''))
		}

		if (activeFilters.tags.length > 0) {
			filtered = filtered.filter(p =>
				p.tags?.some(tag => activeFilters.tags.includes(tag))
			)
		}

		if (activeFilters.price) {
			filtered = filtered.filter(p =>
				p.price >= activeFilters.price!.min && p.price <= activeFilters.price!.max
			)
		}

		if (activeFilters.availability === "in-stock") {
			filtered = filtered.filter(p => p.inStock)
		} else if (activeFilters.availability === "out-of-stock") {
			filtered = filtered.filter(p => !p.inStock)
		}

		// Sorting
		switch (sortOption) {
			case "featured":
				filtered.sort((a, b) => {
					if (a.featured && !b.featured) return -1
					if (!a.featured && b.featured) return 1
					return 0
				})
				break
			case "newest":
				filtered.sort((a, b) => {
					if (a.isNew && !b.isNew) return -1
					if (!a.isNew && b.isNew) return 1
					return 0
				})
				break
			case "price-low-high":
				filtered.sort((a, b) => a.price - b.price)
				break
			case "price-high-low":
				filtered.sort((a, b) => b.price - a.price)
				break
		}

		setFilteredProducts(filtered)
	}, [products, activeCategory, activeFilters, sortOption])

	// Filter sidebar component
	const FilterSidebar = () => (
		<div className="space-y-6">
			<div>
				<h3 className="font-semibold mb-3">Filters</h3>
				<Button
					variant="outline"
					size="sm"
					onClick={() => {
						setActiveFilters({
							categories: [],
							subcategories: [],
							tags: [],
							price: null,
							availability: null,
						})
						setSortOption("featured")
					}}
					className="w-full"
				>
					Clear All Filters
				</Button>
			</div>

			<Separator />

			<Accordion type="multiple" className="w-full">
				{/* Categories */}
				<AccordionItem value="categories">
					<AccordionTrigger>Categories</AccordionTrigger>
					<AccordionContent>
						<div className="space-y-2">
							{categories.map(category => (
								<div key={category} className="flex items-center space-x-2">
									<Checkbox
										id={`category-${category}`}
										checked={activeFilters.categories.includes(category)}
										onCheckedChange={(checked) => {
											if (checked) {
												setActiveFilters(prev => ({
													...prev,
													categories: [...prev.categories, category]
												}))
											} else {
												setActiveFilters(prev => ({
													...prev,
													categories: prev.categories.filter(c => c !== category)
												}))
											}
										}}
									/>
									<Label
										htmlFor={`category-${category}`}
										className="text-sm font-normal cursor-pointer"
									>
										{category}
									</Label>
								</div>
							))}
						</div>
					</AccordionContent>
				</AccordionItem>

				{/* Subcategories */}
				{subcategories.length > 0 && (
					<AccordionItem value="subcategories">
						<AccordionTrigger>Subcategories</AccordionTrigger>
						<AccordionContent>
							<div className="space-y-2">
								{subcategories.map(subcategory => (
									<div key={subcategory} className="flex items-center space-x-2">
										<Checkbox
											id={`subcategory-${subcategory}`}
											checked={activeFilters.subcategories.includes(subcategory)}
											onCheckedChange={(checked) => {
												if (checked) {
													setActiveFilters(prev => ({
														...prev,
														subcategories: [...prev.subcategories, subcategory]
													}))
												} else {
													setActiveFilters(prev => ({
														...prev,
														subcategories: prev.subcategories.filter(s => s !== subcategory)
													}))
												}
											}}
										/>
										<Label
											htmlFor={`subcategory-${subcategory}`}
											className="text-sm font-normal cursor-pointer"
										>
											{subcategory}
										</Label>
									</div>
								))}
							</div>
						</AccordionContent>
					</AccordionItem>
				)}

				{/* Tags */}
				{allTags.length > 0 && (
					<AccordionItem value="tags">
						<AccordionTrigger>Tags</AccordionTrigger>
						<AccordionContent>
							<div className="space-y-2">
								{allTags.map(tag => (
									<div key={tag} className="flex items-center space-x-2">
										<Checkbox
											id={`tag-${tag}`}
											checked={activeFilters.tags.includes(tag)}
											onCheckedChange={(checked) => {
												if (checked) {
													setActiveFilters(prev => ({
														...prev,
														tags: [...prev.tags, tag]
													}))
												} else {
													setActiveFilters(prev => ({
														...prev,
														tags: prev.tags.filter(t => t !== tag)
													}))
												}
											}}
										/>
										<Label
											htmlFor={`tag-${tag}`}
											className="text-sm font-normal cursor-pointer"
										>
											{tag}
										</Label>
									</div>
								))}
							</div>
						</AccordionContent>
					</AccordionItem>
				)}

				{/* Availability */}
				<AccordionItem value="availability">
					<AccordionTrigger>Availability</AccordionTrigger>
					<AccordionContent>
						<div className="space-y-2">
							<div className="flex items-center space-x-2">
								<Checkbox
									id="in-stock"
									checked={activeFilters.availability === "in-stock"}
									onCheckedChange={(checked) => {
										setActiveFilters(prev => ({
											...prev,
											availability: checked ? "in-stock" : null
										}))
									}}
								/>
								<Label htmlFor="in-stock" className="text-sm font-normal cursor-pointer">
									In Stock
								</Label>
							</div>
							<div className="flex items-center space-x-2">
								<Checkbox
									id="out-of-stock"
									checked={activeFilters.availability === "out-of-stock"}
									onCheckedChange={(checked) => {
										setActiveFilters(prev => ({
											...prev,
											availability: checked ? "out-of-stock" : null
										}))
									}}
								/>
								<Label htmlFor="out-of-stock" className="text-sm font-normal cursor-pointer">
									Out of Stock
								</Label>
							</div>
						</div>
					</AccordionContent>
				</AccordionItem>
			</Accordion>
		</div>
	)

	// Active filters display
	const ActiveFilters = () => {
		const hasFilters = activeFilters.categories.length > 0 ||
			activeFilters.subcategories.length > 0 ||
			activeFilters.tags.length > 0 ||
			activeFilters.price !== null ||
			activeFilters.availability !== null

		if (!hasFilters) return null

		return (
			<div className="flex flex-wrap gap-2 mb-4">
				{activeFilters.categories.map(category => (
					<Badge key={category} variant="secondary" className="gap-1">
						{category}
						<X
							className="h-3 w-3 cursor-pointer"
							onClick={() => setActiveFilters(prev => ({
								...prev,
								categories: prev.categories.filter(c => c !== category)
							}))}
						/>
					</Badge>
				))}
				{activeFilters.subcategories.map(subcategory => (
					<Badge key={subcategory} variant="secondary" className="gap-1">
						{subcategory}
						<X
							className="h-3 w-3 cursor-pointer"
							onClick={() => setActiveFilters(prev => ({
								...prev,
								subcategories: prev.subcategories.filter(s => s !== subcategory)
							}))}
						/>
					</Badge>
				))}
				{activeFilters.tags.map(tag => (
					<Badge key={tag} variant="secondary" className="gap-1">
						{tag}
						<X
							className="h-3 w-3 cursor-pointer"
							onClick={() => setActiveFilters(prev => ({
								...prev,
								tags: prev.tags.filter(t => t !== tag)
							}))}
						/>
					</Badge>
				))}
				{activeFilters.availability && (
					<Badge variant="secondary" className="gap-1">
						{activeFilters.availability === "in-stock" ? "In Stock" : "Out of Stock"}
						<X
							className="h-3 w-3 cursor-pointer"
							onClick={() => setActiveFilters(prev => ({
								...prev,
								availability: null
							}))}
						/>
					</Badge>
				)}
			</div>
		)
	}

	return (
		<div className="min-h-screen bg-background">
			{/* Breadcrumb */}
			<div className="px-4 sm:px-6 lg:px-8 py-4">
				<Breadcrumb>
					<BreadcrumbList>
						<BreadcrumbItem>
							<BreadcrumbLink href="/">Home</BreadcrumbLink>
						</BreadcrumbItem>
						<BreadcrumbSeparator />
						<BreadcrumbItem>
							<BreadcrumbLink>Catalog</BreadcrumbLink>
						</BreadcrumbItem>
					</BreadcrumbList>
				</Breadcrumb>
			</div>

			<div className="px-4 sm:px-6 lg:px-8 pb-16">
				{/* Header */}
				<div className="mb-8">
					<h1 className="text-4xl font-bold mb-2">Shop Euro Haus</h1>
					<p className="text-muted-foreground">
						Premium merchandise and event tickets for automotive enthusiasts
					</p>
				</div>

				{/* Category Tabs */}
				<div className="mb-8 overflow-x-auto">
					<div className="flex space-x-1 min-w-max">
						<Button
							variant={activeCategory === "all" ? "default" : "ghost"}
							onClick={() => {
								setActiveCategory("all")
								setActiveFilters(prev => ({ ...prev, categories: [] }))
							}}
						>
							All Products
						</Button>
						{categories.map(category => (
							<Button
								key={category}
								variant={activeCategory === category.toLowerCase() ? "default" : "ghost"}
								onClick={() => {
									setActiveCategory(category.toLowerCase())
									setActiveFilters(prev => ({ ...prev, categories: [] }))
								}}
							>
								{category}
							</Button>
						))}
					</div>
				</div>

				{/* Main Content */}
				<div className="flex gap-8">
					{/* Desktop Sidebar */}
					<aside className="hidden lg:block w-64 flex-shrink-0">
						<FilterSidebar />
					</aside>

					{/* Products Grid */}
					<div className="flex-1">
						{/* Toolbar */}
						<div className="flex items-center justify-between mb-6">
							<p className="text-sm text-muted-foreground">
								{filteredProducts.length} {filteredProducts.length === 1 ? 'product' : 'products'}
							</p>

							<div className="flex items-center gap-2">
								{/* Mobile Filter */}
								<Sheet open={mobileFiltersOpen} onOpenChange={setMobileFiltersOpen}>
									<SheetTrigger asChild className="lg:hidden">
										<Button variant="outline" size="sm">
											<Filter className="h-4 w-4 mr-2" />
											Filters
										</Button>
									</SheetTrigger>
									<SheetContent side="left" className="w-[300px] sm:w-[400px]">
										<SheetHeader>
											<SheetTitle>Filters</SheetTitle>
											<SheetDescription>
												Narrow down your product search
											</SheetDescription>
										</SheetHeader>
										<div className="mt-6">
											<FilterSidebar />
										</div>
									</SheetContent>
								</Sheet>

								{/* Sort Dropdown */}
								<Select value={sortOption} onValueChange={setSortOption}>
									<SelectTrigger className="w-[180px]">
										<SelectValue placeholder="Sort by" />
									</SelectTrigger>
									<SelectContent>
										<SelectItem value="featured">Featured</SelectItem>
										<SelectItem value="newest">Newest</SelectItem>
										<SelectItem value="price-low-high">Price: Low to High</SelectItem>
										<SelectItem value="price-high-low">Price: High to Low</SelectItem>
									</SelectContent>
								</Select>
							</div>
						</div>

						{/* Active Filters */}
						<ActiveFilters />

						{/* Products Grid */}
						{filteredProducts.length > 0 ? (
							<div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-6">
								{filteredProducts.map((product) => (
									<ProductCard
										key={product.id}
										id={product.id}
										title={product.title}
										description={product.description}
										price={product.price}
										imageUrl={product.images[0]}
										isNew={product.isNew}
										compareAtPrice={product.compareAtPrice}
										inStock={product.inStock}
									/>
								))}
							</div>
						) : (
							<div className="text-center py-12">
								<p className="text-muted-foreground mb-4">
									No products found matching your criteria
								</p>
								<Button
									variant="outline"
									onClick={() => {
										setActiveFilters({
											categories: [],
											subcategories: [],
											tags: [],
											price: null,
											availability: null,
										})
										setActiveCategory("all")
										setSortOption("featured")
									}}
								>
									Clear all filters
								</Button>
							</div>
						)}
					</div>
				</div>
			</div>
		</div>
	)
}
