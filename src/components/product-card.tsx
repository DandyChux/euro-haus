import { Image } from "./ui/image"
import { Button } from "~/components/ui/button"
import { Badge } from "./ui/badge"
import { ShoppingCart, Heart } from "lucide-react"
import { useState } from "react"
import { cn } from "~/lib/utils"

interface ProductCardProps {
	id: string
	title: string
	description: string
	price: number
	compareAtPrice?: number
	imageUrl: string
	isNew?: boolean
	inStock?: boolean
}

export function ProductCard({
	id,
	title,
	description,
	price,
	compareAtPrice,
	imageUrl,
	isNew = false,
	inStock = true,
}: ProductCardProps) {
	const [isHovered, setIsHovered] = useState(false)
	const [isFavorite, setIsFavorite] = useState(false)

	const handleAddToCart = (e: React.MouseEvent) => {
		e.preventDefault()
		// In a real implementation, this would add the product to the cart
		console.log(`Adding ${title} to cart`)

		// This would typically dispatch to a cart context or Redux store
		// dispatch({ type: 'ADD_TO_CART', payload: { id, title, price, quantity: 1 } })
	}

	const handleToggleFavorite = (e: React.MouseEvent) => {
		e.preventDefault()
		e.stopPropagation()
		setIsFavorite(!isFavorite)
		// In a real implementation, this would add/remove the product from favorites
		console.log(`${isFavorite ? "Removing" : "Adding"} ${title} ${isFavorite ? "from" : "to"} favorites`)
	}

	const discount = compareAtPrice ? Math.round(((compareAtPrice - price) / compareAtPrice) * 100) : 0

	return (
		<a
			href={`/product/${id}`}
			className="group relative flex flex-col overflow-hidden rounded-lg border bg-background transition-all hover:shadow-md"
			onMouseEnter={() => setIsHovered(true)}
			onMouseLeave={() => setIsHovered(false)}
		>
			<div className="relative aspect-square overflow-hidden bg-muted">
				<Image
					src={imageUrl}
					alt={title}
					className={cn("h-full w-full object-cover transition-transform duration-300", isHovered && "scale-105")}
				/>

				{/* Badges */}
				<div className="absolute left-2 top-2 flex flex-col gap-1">
					{isNew && <Badge className="bg-primary text-primary-foreground">New</Badge>}
					{compareAtPrice && discount > 0 && <Badge variant="destructive">-{discount}%</Badge>}
					{!inStock && (
						<Badge variant="outline" className="bg-background/80">
							Out of Stock
						</Badge>
					)}
				</div>

				{/* Favorite Button */}
				<button
					className={cn(
						"absolute right-2 top-2 rounded-full bg-background/80 p-1.5 text-foreground transition-all hover:bg-background",
						isFavorite && "text-red-500",
					)}
					onClick={handleToggleFavorite}
				>
					<Heart className="h-5 w-5" fill={isFavorite ? "currentColor" : "none"} />
				</button>
			</div>

			<div className="flex flex-1 flex-col p-4">
				<h3 className="font-medium line-clamp-1">{title}</h3>
				<p className="mt-1 text-sm text-muted-foreground line-clamp-2">{description}</p>

				<div className="mt-auto pt-4">
					<div className="flex items-center justify-between">
						<div className="flex items-center gap-1">
							<span className="font-bold">${price.toFixed(2)}</span>
							{compareAtPrice && (
								<span className="text-sm text-muted-foreground line-through">${compareAtPrice.toFixed(2)}</span>
							)}
						</div>

						{inStock ? (
							<Button size="sm" className="h-8 w-8 rounded-full p-0" onClick={handleAddToCart}>
								<ShoppingCart className="h-4 w-4" />
								<span className="sr-only">Add to cart</span>
							</Button>
						) : (
							<Button size="sm" variant="outline" disabled>
								Sold Out
							</Button>
						)}
					</div>
				</div>
			</div>
		</a>
	)
}
