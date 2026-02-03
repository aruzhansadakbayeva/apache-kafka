from pyspark.sql import SparkSession
from pyspark.sql.functions import col, count, row_number, struct, to_json
from pyspark.sql.window import Window

spark = SparkSession.builder.appName("cart-analytics").getOrCreate()

input_path = "hdfs://namenode:8020/data/cart/dt=*/*.jsonl"
out_path   = "hdfs://namenode:8020/data/reco/dt=latest"

df = spark.read.json(input_path)

# top-3 products per user by frequency
agg = df.groupBy("user_id", "product_id").agg(count("*").alias("cnt"))

w = Window.partitionBy("user_id").orderBy(col("cnt").desc())
top = agg.withColumn("rn", row_number().over(w)).where(col("rn") <= 3)

# pack into one record per user
reco = top.groupBy("user_id").agg(
    to_json(struct(
        col("user_id").alias("user_id"),
        # простая форма: массив пар product_id/cnt сделаем через collect_list
    )).alias("json")
)

# проще: запишем как обычный json (spark сам сделает строки)
top.coalesce(1).write.mode("overwrite").json(out_path)

spark.stop()
