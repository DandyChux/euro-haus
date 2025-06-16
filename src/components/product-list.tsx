import React, { useState, useEffect } from 'react';
import type { Product } from '~/lib/interfaces/product';
import { Image } from './ui/image';

const ProductList: React.FC = () => {
	const [products, setProducts] = useState<Product[]>([]);
	const [loading, setLoading] = useState<boolean>(true);
	const [error, setError] = useState<string | null>(null);

	useEffect(() => {
		const fetchProducts = async () => {
			try {
				const response = await fetch(`${import.meta.env.VITE_API_URL}/products`);

				if (!response.ok) {
					throw new Error(`HTTP error! Status: ${response.status}`);
				}

				const data: Product[] = await response.json();
				setProducts(data);
			} catch (err) {
				setError(err instanceof Error ? err.message : 'Unknown error');
			} finally {
				setLoading(false);
			}
		};

		fetchProducts();
	}, []);

	if (loading) return <div className="loading">Loading products...</div>;
	if (error) return <div className="error">Error: {error}</div>;

	return (
		<div className="product-grid">
			{products.map(product => (
				<div key={product.id} className="product-card">
					{product.images.length > 0 && (
						<Image
							src={product.images[0]}
							alt={product.name}
							className="product-image"
							width={200}
							height={150}
						/>
					)}
					<h3>{product.name}</h3>
					<p className="description">{product.description}</p>
					<div className="price">
						${(product.price / 100).toFixed(2)} {product.currency.toUpperCase()}
					</div>
					<button
						onClick={() => {/* Open checkout modal */ }}
						className="buy-button"
					>
						Buy Now
					</button>
				</div>
			))}
		</div>
	);
};

export default ProductList;
