import React, { useEffect, useState } from 'react';
import { createFileRoute } from '@tanstack/react-router'
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbSeparator } from '~/components/ui/breadcrumb'
import { Button } from '~/components/ui/button';
import { Badge } from '~/components/ui/badge';
import { ChevronDown, Filter, SlidersHorizontal, X } from 'lucide-react';
import { Separator } from '~/components/ui/separator';
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '~/components/ui/accordion';
import { Checkbox } from '~/components/ui/checkbox';
import { Label } from '~/components/ui/label';
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from '~/components/ui/sheet';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '~/components/ui/select';
import { ProductCard } from '~/components/product-card';

export const Route = createFileRoute('/catalog')({
	component: RouteComponent,
})

interface Product {
	id: string
	title: string
	description: string
	price: number
	compareAtPrice?: number
	imageUrl: string
	category: string
	subcategory: string
	tags: string[]
	isNew: boolean
	isFeatured: boolean
	inStock: boolean
}

function RouteComponent() {
	const [products, setProducts] = useState<Product[]>([])
	const [filteredProducts, setFilteredProducts] = useState<Product[]>([])
	const [isLoading, setIsLoading] = useState(true)
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

	// Fetch products from Shopify (mock implementation)
	// TODO: Implement actual fetch from Shopify API
	useEffect(() => {
		const fetchProducts = async () => {
			setIsLoading(true)
			try {
				// In a real implementation, this would be a fetch to your Shopify storefront API
				// For now, we'll use mock data
				const mockProducts: Product[] = [
					{
						id: "1",
						title: "Euro Haus T-Shirt",
						description: "Premium cotton t-shirt with Euro Haus logo",
						price: 29.99,
						imageUrl: "/placeholder.svg?height=400&width=400",
						category: "Apparel",
						subcategory: "T-Shirts",
						tags: ["clothing", "casual"],
						isNew: true,
						isFeatured: true,
						inStock: true,
					},
					{
						id: "2",
						title: "Rally Cap",
						description: "Adjustable cap with embroidered logo",
						price: 24.99,
						imageUrl: "/placeholder.svg?height=400&width=400",
						category: "Apparel",
						subcategory: "Headwear",
						tags: ["clothing", "accessories"],
						isNew: false,
						isFeatured: true,
						inStock: true,
					},
					{
						id: "3",
						title: "Orlando Rally 2025 Ticket",
						description: "General admission to our flagship event",
						price: 149.99,
						imageUrl: "/placeholder.svg?height=400&width=400",
						category: "Events",
						subcategory: "Tickets",
						tags: ["event", "rally"],
						isNew: true,
						isFeatured: true,
						inStock: true,
					},
					{
						id: "4",
						title: "Euro Haus Hoodie",
						description: "Warm hoodie with front pocket and logo",
						price: 59.99,
						imageUrl: "/placeholder.svg?height=400&width=400",
						category: "Apparel",
						subcategory: "Hoodies",
						tags: ["clothing", "winter"],
						isNew: false,
						isFeatured: false,
						inStock: true,
					},
					{
						id: "5",
						title: "Car Detailing Kit",
						description: "Complete kit for professional-level detailing",
						price: 89.99,
						imageUrl: "/placeholder.svg?height=400&width=400",
						category: "Accessories",
						subcategory: "Car Care",
						tags: ["maintenance", "cleaning"],
						isNew: true,
						isFeatured: false,
						inStock: true,
					},
					{
						id: "6",
						title: "Track Day Experience",
						description: "Full day at the track with professional instruction",
						price: 299.99,
						imageUrl: "/placeholder.svg?height=400&width=400",
						category: "Events",
						subcategory: "Experiences",
						tags: ["event", "track"],
						isNew: false,
						isFeatured: true,
						inStock: true,
					},
					{
						id: "7",
						title: "Euro Haus Sticker Pack",
						description: "Set of 5 vinyl stickers with various designs",
						price: 12.99,
						imageUrl: "/placeholder.svg?height=400&width=400",
						category: "Accessories",
						subcategory: "Stickers",
						tags: ["accessories", "decoration"],
						isNew: false,
						isFeatured: false,
						inStock: true,
					},
					{
						id: "8",
						title: "Limited Edition Poster",
						description: "Commemorative poster from our last major event",
						price: 19.99,
						compareAtPrice: 29.99,
						imageUrl: "/placeholder.svg?height=400&width=400",
						category: "Accessories",
						subcategory: "Wall Art",
						tags: ["home", "decoration"],
						isNew: false,
						isFeatured: false,
						inStock: false,
					},
					{
						id: "9",
						title: "Summer Car Meet Ticket",
						description: "Entry to our summer gathering in Miami",
						price: 49.99,
						imageUrl: "/placeholder.svg?height=400&width=400",
						category: "Events",
						subcategory: "Tickets",
						tags: ["event", "meet"],
						isNew: true,
						isFeatured: true,
						inStock: true,
					},
					{
						id: "10",
						title: "Euro Haus Tumbler",
						description: "Insulated tumbler with logo, perfect for road trips",
						price: 34.99,
						imageUrl: "/placeholder.svg?height=400&width=400",
						category: "Accessories",
						subcategory: "Drinkware",
						tags: ["lifestyle", "travel"],
						isNew: true,
						isFeatured: false,
						inStock: true,
					},
					{
						id: "11",
						title: "Performance Driving Gloves",
						description: "Leather driving gloves for optimal grip and comfort",
						price: 79.99,
						imageUrl: "/placeholder.svg?height=400&width=400",
						category: "Accessories",
						subcategory: "Driving Gear",
						tags: ["driving", "performance"],
						isNew: false,
						isFeatured: true,
						inStock: true,
					},
					{
						id: "12",
						title: "Euro Haus Polo Shirt",
						description: "Embroidered polo shirt with moisture-wicking fabric",
						price: 39.99,
						imageUrl: "/placeholder.svg?height=400&width=400",
						category: "Apparel",
						subcategory: "Shirts",
						tags: ["clothing", "premium"],
						isNew: false,
						isFeatured: false,
						inStock: true,
					},
				]

				setProducts(mockProducts)
				setFilteredProducts(mockProducts)
			} catch (error) {
				console.error("Error fetching products:", error)
			} finally {
				setIsLoading(false)
			}
		}

		fetchProducts()
	}, [])

	// Apply filters and sorting
	useEffect(() => {
		let result = [...products]

		// Filter by category
		if (activeCategory !== "all") {
			result = result.filter((product) => product.category === activeCategory)
		}

		// Apply active filters
		if (activeFilters.categories.length > 0) {
			result = result.filter((product) => activeFilters.categories.includes(product.category))
		}

		if (activeFilters.subcategories.length > 0) {
			result = result.filter((product) => activeFilters.subcategories.includes(product.subcategory))
		}

		if (activeFilters.tags.length > 0) {
			result = result.filter((product) => product.tags.some((tag) => activeFilters.tags.includes(tag)))
		}

		if (activeFilters.price) {
			result = result.filter(
				(product) => product.price >= activeFilters.price!.min && product.price <= activeFilters.price!.max,
			)
		}

		if (activeFilters.availability === "in-stock") {
			result = result.filter((product) => product.inStock)
		} else if (activeFilters.availability === "out-of-stock") {
			result = result.filter((product) => !product.inStock)
		}

		// Apply sorting
		switch (sortOption) {
			case "price-low":
				result.sort((a, b) => a.price - b.price)
				break
			case "price-high":
				result.sort((a, b) => b.price - a.price)
				break
			case "newest":
				result.sort((a, b) => (a.isNew === b.isNew ? 0 : a.isNew ? -1 : 1))
				break
			case "featured":
			default:
				result.sort((a, b) => (a.isFeatured === b.isFeatured ? 0 : a.isFeatured ? -1 : 1))
				break
		}

		setFilteredProducts(result)
	}, [products, activeCategory, activeFilters, sortOption])

	// Get unique categories for filter options
	const categories = Array.from(new Set(products.map((product) => product.category)))
	const subcategories = Array.from(new Set(products.map((product) => product.subcategory)))
	const tags = Array.from(new Set(products.flatMap((product) => product.tags)))

	// Toggle filter function
	const toggleFilter = (type: "categories" | "subcategories" | "tags", value: string) => {
		setActiveFilters((prev) => {
			const current = [...prev[type]]
			const index = current.indexOf(value)

			if (index === -1) {
				current.push(value)
			} else {
				current.splice(index, 1)
			}

			return {
				...prev,
				[type]: current,
			}
		})
	}

	// Clear all filters
	const clearFilters = () => {
		setActiveFilters({
			categories: [],
			subcategories: [],
			tags: [],
			price: null,
			availability: null,
		})
		setActiveCategory("all")
	}

	// Handle category change
	const handleCategoryChange = (category: string) => {
		setActiveCategory(category)
		setMobileFiltersOpen(false)
	}

	return (
		<section className='px-4 py-8 md:px-6'>
			<Breadcrumb className='mb-6'>
				{/* TODO: Use variables to create breadcrumb items */}
				<BreadcrumbList>
					<BreadcrumbItem>
						<BreadcrumbLink href='/'>Home</BreadcrumbLink>
					</BreadcrumbItem>
					<BreadcrumbSeparator />
					<BreadcrumbItem>
						<BreadcrumbLink href='/catalog'>Catalog</BreadcrumbLink>
					</BreadcrumbItem>
					<BreadcrumbSeparator />
					<BreadcrumbItem>
						<BreadcrumbLink href='#' isCurrentPage>
							{activeCategory === 'all' ? 'All Products' : activeCategory}
						</BreadcrumbLink>
					</BreadcrumbItem>
				</BreadcrumbList>
			</Breadcrumb>

			<div className='mb-8'>
				<h1 className='text-3xl font-bold tracking-tight md:text-4xl'>
					{activeCategory === 'all' ? 'Shop All Products' : activeCategory}
				</h1>
				<p className='mt-2 text-muted-foreground'>
					Browse our colelction of exclusive merchandise, event tickets, and accessories.
				</p>
			</div>

			<div className='mb-6 flex flex-wrap gap-2'>
				<Button
					variant={activeCategory === 'all' ? 'default' : 'outline'}
					size='sm'
					onClick={() => handleCategoryChange('all')}
				>
					All Products
				</Button>
				{categories.map((category) => (
					<Button
						key={category}
						variant={activeCategory === category ? "default" : "outline"}
						size="sm"
						onClick={() => handleCategoryChange(category)}
					>
						{category}
					</Button>
				))}
			</div>

			{/* Active Filters */}
			{(activeFilters.categories.length > 0 ||
				activeFilters.subcategories.length > 0 ||
				activeFilters.tags.length > 0 ||
				activeFilters.price ||
				activeFilters.availability) && (
					<div className="mb-6 flex flex-wrap items-center gap-2">
						<span className="text-sm font-medium">Active Filters:</span>
						{activeFilters.categories.map((cat) => (
							<Badge key={`cat-${cat}`} variant="secondary" className="flex items-center gap-1">
								{cat}
								<button onClick={() => toggleFilter("categories", cat)}>
									<X className="h-3 w-3" />
								</button>
							</Badge>
						))}
						{activeFilters.subcategories.map((subcat) => (
							<Badge key={`subcat-${subcat}`} variant="secondary" className="flex items-center gap-1">
								{subcat}
								<button onClick={() => toggleFilter("subcategories", subcat)}>
									<X className="h-3 w-3" />
								</button>
							</Badge>
						))}
						{activeFilters.tags.map((tag) => (
							<Badge key={`tag-${tag}`} variant="secondary" className="flex items-center gap-1">
								{tag}
								<button onClick={() => toggleFilter("tags", tag)}>
									<X className="h-3 w-3" />
								</button>
							</Badge>
						))}
						{activeFilters.price && (
							<Badge variant="secondary" className="flex items-center gap-1">
								${activeFilters.price.min} - ${activeFilters.price.max}
								<button onClick={() => setActiveFilters((prev) => ({ ...prev, price: null }))}>
									<X className="h-3 w-3" />
								</button>
							</Badge>
						)}
						{activeFilters.availability && (
							<Badge variant="secondary" className="flex items-center gap-1">
								{activeFilters.availability === "in-stock" ? "In Stock" : "Out of Stock"}
								<button onClick={() => setActiveFilters((prev) => ({ ...prev, availability: null }))}>
									<X className="h-3 w-3" />
								</button>
							</Badge>
						)}
						<Button variant="ghost" size="sm" onClick={clearFilters} className="ml-2">
							Clear All
						</Button>
					</div>
				)}

			<div className="grid grid-cols-1 gap-6 lg:grid-cols-4">
				{/* Filters - Desktop */}
				<div className="hidden lg:block">
					<div className="sticky top-24 space-y-6">
						<div className="flex items-center justify-between">
							<h3 className="text-lg font-semibold">Filters</h3>
							<Button variant="ghost" size="sm" onClick={clearFilters}>
								Clear All
							</Button>
						</div>
						<Separator />

						<Accordion type="multiple" defaultValue={["categories", "price", "availability"]}>
							<AccordionItem value="categories">
								<AccordionTrigger>Categories</AccordionTrigger>
								<AccordionContent>
									<div className="space-y-2">
										{categories.map((category) => (
											<div key={category} className="flex items-center space-x-2">
												<Checkbox
													id={`category-${category}`}
													checked={activeFilters.categories.includes(category)}
													onCheckedChange={() => toggleFilter("categories", category)}
												/>
												<Label htmlFor={`category-${category}`} className="text-sm">
													{category}
												</Label>
											</div>
										))}
									</div>
								</AccordionContent>
							</AccordionItem>

							<AccordionItem value="subcategories">
								<AccordionTrigger>Product Types</AccordionTrigger>
								<AccordionContent>
									<div className="space-y-2">
										{subcategories.map((subcategory) => (
											<div key={subcategory} className="flex items-center space-x-2">
												<Checkbox
													id={`subcategory-${subcategory}`}
													checked={activeFilters.subcategories.includes(subcategory)}
													onCheckedChange={() => toggleFilter("subcategories", subcategory)}
												/>
												<Label htmlFor={`subcategory-${subcategory}`} className="text-sm">
													{subcategory}
												</Label>
											</div>
										))}
									</div>
								</AccordionContent>
							</AccordionItem>

							<AccordionItem value="tags">
								<AccordionTrigger>Tags</AccordionTrigger>
								<AccordionContent>
									<div className="space-y-2">
										{tags.map((tag) => (
											<div key={tag} className="flex items-center space-x-2">
												<Checkbox
													id={`tag-${tag}`}
													checked={activeFilters.tags.includes(tag)}
													onCheckedChange={() => toggleFilter("tags", tag)}
												/>
												<Label htmlFor={`tag-${tag}`} className="text-sm">
													{tag}
												</Label>
											</div>
										))}
									</div>
								</AccordionContent>
							</AccordionItem>

							<AccordionItem value="price">
								<AccordionTrigger>Price Range</AccordionTrigger>
								<AccordionContent>
									<div className="space-y-4">
										<div className="flex items-center space-x-2">
											<Checkbox
												id="price-under-25"
												checked={activeFilters.price?.max === 25}
												onCheckedChange={() =>
													setActiveFilters((prev) => ({
														...prev,
														price: prev.price?.max === 25 ? null : { min: 0, max: 25 },
													}))
												}
											/>
											<Label htmlFor="price-under-25" className="text-sm">
												Under $25
											</Label>
										</div>
										<div className="flex items-center space-x-2">
											<Checkbox
												id="price-25-50"
												checked={activeFilters.price?.min === 25 && activeFilters.price?.max === 50}
												onCheckedChange={() =>
													setActiveFilters((prev) => ({
														...prev,
														price: prev.price?.min === 25 && prev.price?.max === 50 ? null : { min: 25, max: 50 },
													}))
												}
											/>
											<Label htmlFor="price-25-50" className="text-sm">
												$25 - $50
											</Label>
										</div>
										<div className="flex items-center space-x-2">
											<Checkbox
												id="price-50-100"
												checked={activeFilters.price?.min === 50 && activeFilters.price?.max === 100}
												onCheckedChange={() =>
													setActiveFilters((prev) => ({
														...prev,
														price: prev.price?.min === 50 && prev.price?.max === 100 ? null : { min: 50, max: 100 },
													}))
												}
											/>
											<Label htmlFor="price-50-100" className="text-sm">
												$50 - $100
											</Label>
										</div>
										<div className="flex items-center space-x-2">
											<Checkbox
												id="price-over-100"
												checked={activeFilters.price?.min === 100 && activeFilters.price?.max === 1000}
												onCheckedChange={() =>
													setActiveFilters((prev) => ({
														...prev,
														price:
															prev.price?.min === 100 && prev.price?.max === 1000 ? null : { min: 100, max: 1000 },
													}))
												}
											/>
											<Label htmlFor="price-over-100" className="text-sm">
												Over $100
											</Label>
										</div>
									</div>
								</AccordionContent>
							</AccordionItem>

							<AccordionItem value="availability">
								<AccordionTrigger>Availability</AccordionTrigger>
								<AccordionContent>
									<div className="space-y-2">
										<div className="flex items-center space-x-2">
											<Checkbox
												id="in-stock"
												checked={activeFilters.availability === "in-stock"}
												onCheckedChange={() =>
													setActiveFilters((prev) => ({
														...prev,
														availability: prev.availability === "in-stock" ? null : "in-stock",
													}))
												}
											/>
											<Label htmlFor="in-stock" className="text-sm">
												In Stock
											</Label>
										</div>
										<div className="flex items-center space-x-2">
											<Checkbox
												id="out-of-stock"
												checked={activeFilters.availability === "out-of-stock"}
												onCheckedChange={() =>
													setActiveFilters((prev) => ({
														...prev,
														availability: prev.availability === "out-of-stock" ? null : "out-of-stock",
													}))
												}
											/>
											<Label htmlFor="out-of-stock" className="text-sm">
												Out of Stock
											</Label>
										</div>
									</div>
								</AccordionContent>
							</AccordionItem>
						</Accordion>
					</div>
				</div>

				{/* Product Grid */}
				<div className="lg:col-span-3">
					{/* Mobile Filter and Sort Controls */}
					<div className="mb-6 flex items-center justify-between lg:hidden">
						<Sheet open={mobileFiltersOpen} onOpenChange={setMobileFiltersOpen}>
							<SheetTrigger asChild>
								<Button variant="outline" size="sm" className="flex items-center gap-2">
									<Filter className="h-4 w-4" />
									Filters
								</Button>
							</SheetTrigger>
							<SheetContent side="left" className="w-[300px] sm:w-[400px]">
								<SheetHeader>
									<SheetTitle>Filters</SheetTitle>
									<SheetDescription>Narrow down products by applying filters</SheetDescription>
								</SheetHeader>
								<div className="mt-6 space-y-6">
									<Accordion type="multiple" defaultValue={["categories", "price", "availability"]}>
										<AccordionItem value="categories">
											<AccordionTrigger>Categories</AccordionTrigger>
											<AccordionContent>
												<div className="space-y-2">
													{categories.map((category) => (
														<div key={category} className="flex items-center space-x-2">
															<Checkbox
																id={`mobile-category-${category}`}
																checked={activeFilters.categories.includes(category)}
																onCheckedChange={() => toggleFilter("categories", category)}
															/>
															<Label htmlFor={`mobile-category-${category}`} className="text-sm">
																{category}
															</Label>
														</div>
													))}
												</div>
											</AccordionContent>
										</AccordionItem>

										<AccordionItem value="subcategories">
											<AccordionTrigger>Product Types</AccordionTrigger>
											<AccordionContent>
												<div className="space-y-2">
													{subcategories.map((subcategory) => (
														<div key={subcategory} className="flex items-center space-x-2">
															<Checkbox
																id={`mobile-subcategory-${subcategory}`}
																checked={activeFilters.subcategories.includes(subcategory)}
																onCheckedChange={() => toggleFilter("subcategories", subcategory)}
															/>
															<Label htmlFor={`mobile-subcategory-${subcategory}`} className="text-sm">
																{subcategory}
															</Label>
														</div>
													))}
												</div>
											</AccordionContent>
										</AccordionItem>

										<AccordionItem value="tags">
											<AccordionTrigger>Tags</AccordionTrigger>
											<AccordionContent>
												<div className="space-y-2">
													{tags.map((tag) => (
														<div key={tag} className="flex items-center space-x-2">
															<Checkbox
																id={`mobile-tag-${tag}`}
																checked={activeFilters.tags.includes(tag)}
																onCheckedChange={() => toggleFilter("tags", tag)}
															/>
															<Label htmlFor={`mobile-tag-${tag}`} className="text-sm">
																{tag}
															</Label>
														</div>
													))}
												</div>
											</AccordionContent>
										</AccordionItem>

										<AccordionItem value="price">
											<AccordionTrigger>Price Range</AccordionTrigger>
											<AccordionContent>
												<div className="space-y-4">
													<div className="flex items-center space-x-2">
														<Checkbox
															id="mobile-price-under-25"
															checked={activeFilters.price?.max === 25}
															onCheckedChange={() =>
																setActiveFilters((prev) => ({
																	...prev,
																	price: prev.price?.max === 25 ? null : { min: 0, max: 25 },
																}))
															}
														/>
														<Label htmlFor="mobile-price-under-25" className="text-sm">
															Under $25
														</Label>
													</div>
													<div className="flex items-center space-x-2">
														<Checkbox
															id="mobile-price-25-50"
															checked={activeFilters.price?.min === 25 && activeFilters.price?.max === 50}
															onCheckedChange={() =>
																setActiveFilters((prev) => ({
																	...prev,
																	price:
																		prev.price?.min === 25 && prev.price?.max === 50 ? null : { min: 25, max: 50 },
																}))
															}
														/>
														<Label htmlFor="mobile-price-25-50" className="text-sm">
															$25 - $50
														</Label>
													</div>
													<div className="flex items-center space-x-2">
														<Checkbox
															id="mobile-price-50-100"
															checked={activeFilters.price?.min === 50 && activeFilters.price?.max === 100}
															onCheckedChange={() =>
																setActiveFilters((prev) => ({
																	...prev,
																	price:
																		prev.price?.min === 50 && prev.price?.max === 100 ? null : { min: 50, max: 100 },
																}))
															}
														/>
														<Label htmlFor="mobile-price-50-100" className="text-sm">
															$50 - $100
														</Label>
													</div>
													<div className="flex items-center space-x-2">
														<Checkbox
															id="mobile-price-over-100"
															checked={activeFilters.price?.min === 100 && activeFilters.price?.max === 1000}
															onCheckedChange={() =>
																setActiveFilters((prev) => ({
																	...prev,
																	price:
																		prev.price?.min === 100 && prev.price?.max === 1000
																			? null
																			: { min: 100, max: 1000 },
																}))
															}
														/>
														<Label htmlFor="mobile-price-over-100" className="text-sm">
															Over $100
														</Label>
													</div>
												</div>
											</AccordionContent>
										</AccordionItem>

										<AccordionItem value="availability">
											<AccordionTrigger>Availability</AccordionTrigger>
											<AccordionContent>
												<div className="space-y-2">
													<div className="flex items-center space-x-2">
														<Checkbox
															id="mobile-in-stock"
															checked={activeFilters.availability === "in-stock"}
															onCheckedChange={() =>
																setActiveFilters((prev) => ({
																	...prev,
																	availability: prev.availability === "in-stock" ? null : "in-stock",
																}))
															}
														/>
														<Label htmlFor="mobile-in-stock" className="text-sm">
															In Stock
														</Label>
													</div>
													<div className="flex items-center space-x-2">
														<Checkbox
															id="mobile-out-of-stock"
															checked={activeFilters.availability === "out-of-stock"}
															onCheckedChange={() =>
																setActiveFilters((prev) => ({
																	...prev,
																	availability: prev.availability === "out-of-stock" ? null : "out-of-stock",
																}))
															}
														/>
														<Label htmlFor="mobile-out-of-stock" className="text-sm">
															Out of Stock
														</Label>
													</div>
												</div>
											</AccordionContent>
										</AccordionItem>
									</Accordion>

									<div className="flex justify-between pt-4">
										<Button variant="outline" onClick={clearFilters}>
											Clear All
										</Button>
										<Button onClick={() => setMobileFiltersOpen(false)}>Apply Filters</Button>
									</div>
								</div>
							</SheetContent>
						</Sheet>

						<div className="flex items-center gap-2">
							<span className="text-sm">Sort by:</span>
							<Select value={sortOption} onValueChange={setSortOption}>
								<SelectTrigger className="w-[160px]">
									<SelectValue placeholder="Sort by" />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="featured">Featured</SelectItem>
									<SelectItem value="newest">Newest</SelectItem>
									<SelectItem value="price-low">Price: Low to High</SelectItem>
									<SelectItem value="price-high">Price: High to Low</SelectItem>
								</SelectContent>
							</Select>
						</div>
					</div>

					{/* Desktop Sort */}
					<div className="mb-6 hidden items-center justify-end lg:flex">
						<div className="flex items-center gap-2">
							<span className="text-sm">Sort by:</span>
							<Select value={sortOption} onValueChange={setSortOption}>
								<SelectTrigger className="w-[180px]">
									<SelectValue placeholder="Sort by" />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="featured">Featured</SelectItem>
									<SelectItem value="newest">Newest</SelectItem>
									<SelectItem value="price-low">Price: Low to High</SelectItem>
									<SelectItem value="price-high">Price: High to Low</SelectItem>
								</SelectContent>
							</Select>
						</div>
					</div>

					{/* Products */}
					{isLoading ? (
						<div className="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3">
							{Array.from({ length: 6 }).map((_, index) => (
								<div key={index} className="h-[350px] animate-pulse rounded-lg bg-muted"></div>
							))}
						</div>
					) : filteredProducts.length === 0 ? (
						<div className="flex flex-col items-center justify-center py-12 text-center">
							<SlidersHorizontal className="mb-4 h-12 w-12 text-muted-foreground" />
							<h3 className="text-lg font-semibold">No products found</h3>
							<p className="mt-2 text-muted-foreground">Try adjusting your filters or browse all products.</p>
							<Button onClick={clearFilters} className="mt-4">
								Clear Filters
							</Button>
						</div>
					) : (
						<div className="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3">
							{filteredProducts.map((product) => (
								<ProductCard
									key={product.id}
									id={product.id}
									title={product.title}
									description={product.description}
									price={product.price}
									compareAtPrice={product.compareAtPrice}
									imageUrl={product.imageUrl}
									isNew={product.isNew}
									inStock={product.inStock}
								/>
							))}
						</div>
					)}

					{/* Pagination */}
					{filteredProducts.length > 0 && (
						<div className="mt-12 flex justify-center">
							<div className="flex items-center space-x-2">
								<Button variant="outline" size="icon" disabled>
									<ChevronDown className="h-4 w-4 rotate-90" />
								</Button>
								<Button variant="outline" size="sm" className="h-8 w-8">
									1
								</Button>
								<Button variant="ghost" size="sm" className="h-8 w-8">
									2
								</Button>
								<Button variant="ghost" size="sm" className="h-8 w-8">
									3
								</Button>
								<Button variant="outline" size="icon">
									<ChevronDown className="h-4 w-4 -rotate-90" />
								</Button>
							</div>
						</div>
					)}
				</div>
			</div>
		</section>
	)
}
